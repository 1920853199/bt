package controller

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"bt/logger"
)

type QueueHandshakeElement struct {
	msgType uint32
	packet  []byte
	buffer  *[MaxMessageSize]byte
	source  *net.UDPAddr
}

type QueueInboundElement struct {
	dropped int32
	mutex   sync.Mutex
	buffer  *[MaxMessageSize]byte
	packet  []byte
	counter uint64
	keyPair *KeyPair
}

func (elem *QueueInboundElement) Drop() {
	atomic.StoreInt32(&elem.dropped, AtomicTrue)
}

func (elem *QueueInboundElement) IsDropped() bool {
	return atomic.LoadInt32(&elem.dropped) == AtomicTrue
}

func (device *Device) addToInboundQueue(
	queue chan *QueueInboundElement,
	element *QueueInboundElement,
) {
	for {
		select {
		case queue <- element:
			return
		default:
			select {
			case old := <-queue:
				old.Drop()
			default:
			}
		}
	}
}

func (device *Device) addToDecryptionQueue(
	queue chan *QueueInboundElement,
	element *QueueInboundElement,
) {
	for {
		select {
		case queue <- element:
			return
		default:
			select {
			case old := <-queue:
				// drop & release to potential consumer
				old.Drop()
				old.mutex.Unlock()
			default:
			}
		}
	}
}

func (device *Device) addToHandshakeQueue(
	queue chan QueueHandshakeElement,
	element QueueHandshakeElement,
) {
	for {
		select {
		case queue <- element:
			return
		default:
			select {
			case elem := <-queue:
				device.PutMessageBuffer(elem.buffer)
			default:
			}
		}
	}
}

func (device *Device) RoutineReceiveIncomming() {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover RoutineReceiveIncomming err:", err))
		}
	}()

	logger.Wlog.SaveDebugLog("Routine, receive incomming, started")

	for {
		// wait for new conn
		logger.Wlog.SaveDebugLog("Waiting for udp socket")

		select {
		case <-device.signal.stop:
			return
		case <-device.signal.newUDPConn:
			// fetch connection
			device.net.mutex.RLock()
			conn := device.net.conn
			device.net.mutex.RUnlock()
			if conn == nil {
				continue
			}

			logger.Wlog.SaveDebugLog("Listening for inbound packets")
			// receive datagrams until conn is closed

			go device.handleUDP(conn)
		}
	}
}

func (device *Device) handleUDP(conn *net.UDPConn) {
	buffer := device.GetMessageBuffer()

	for {
		size, raddr, err := conn.ReadFromUDP(buffer[:])
		if err != nil {
			if strings.Contains(err.Error(), "message too long") {
				logger.Wlog.SaveErrLog("读取UDP失败:" + err.Error())
				continue
			} else {
				logger.Wlog.SaveErrLog("UDP退出:" + err.Error())
				return
			}
		}

		logger.Wlog.SaveErrLog(fmt.Sprintln("收到 UDP 数据:", size, err))
		if size < MinMessageSize {
			continue
		}

		//DownloadFlowNum += size
		// check size of packet
		packet := buffer[:size]
		msgType := binary.LittleEndian.Uint32(packet[:4])

		IntervalStartTime = time.Now().Unix()
		if !firstConnSuccess {
			go sendStatus(1)
			firstConnSuccess = true
		}

		var okay bool
		switch msgType {
		// check if transport
		case MessageTransportType:
			// check size
			if len(packet) < MessageTransportType {
				continue
			}

			// lookup key pair
			receiver := binary.LittleEndian.Uint32(
				packet[MessageTransportOffsetReceiver:MessageTransportOffsetCounter],
			)
			value := device.indices.Lookup(receiver)
			keyPair := value.keyPair
			if keyPair == nil {
				continue
			}

			// check key-pair expiry
			if keyPair.created.Add(RejectAfterTime).Before(time.Now()) {
				continue
			}

			// create work element
			peer := value.peer
			elem := &QueueInboundElement{
				packet:  packet,
				buffer:  buffer,
				keyPair: keyPair,
				dropped: AtomicFalse,
			}
			elem.mutex.Lock()

			// add to decryption queues
			device.addToDecryptionQueue(device.queue.decryption, elem)
			device.addToInboundQueue(peer.queue.inbound, elem)
			buffer = device.GetMessageBuffer()
			continue

			// otherwise it is a handshake related packet

		case MessageInitiationType:
			okay = len(packet) == MessageInitiationSize
		case MessageResponseType:
			okay = len(packet) == MessageResponseSize
		case MessageCookieReplyType:
			okay = len(packet) == MessageCookieReplySize
		}

		if okay {
			device.addToHandshakeQueue(
				device.queue.handshake,
				QueueHandshakeElement{
					msgType: msgType,
					buffer:  buffer,
					packet:  packet,
					source:  raddr,
				},
			)

			buffer = device.GetMessageBuffer()
		}
	}
}

func (device *Device) RoutineDecryption() {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover RoutineDecryption err:", err))
		}
	}()

	var nonce [chacha20poly1305.NonceSize]byte

	logger.Wlog.SaveDebugLog("Routine, decryption, started for device")

	for {
		select {
		case <-device.signal.stop:
			logger.Wlog.SaveDebugLog("Routine, decryption worker, stopped")
			return
		case elem := <-device.queue.decryption:
			// check if dropped
			if elem.IsDropped() {
				continue
			}

			// split message into fields
			counter := elem.packet[MessageTransportOffsetCounter:MessageTransportOffsetContent]
			content := elem.packet[MessageTransportOffsetContent:]

			// decrypt and release to consumer
			var err error
			copy(nonce[4:], counter)
			elem.counter = binary.LittleEndian.Uint64(counter)
			elem.packet, err = elem.keyPair.receive.Open(
				elem.buffer[:0],
				nonce[:],
				content,
				nil,
			)
			if err != nil {
				elem.Drop()
			}
			elem.mutex.Unlock()
		}
	}
}

/* Handles incomming packets related to handshake
 */
func (device *Device) RoutineHandshake() {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover RoutineHandshake err:", err))
		}
	}()

	logger.Wlog.SaveDebugLog("Routine, handshake routine, started for device")
	var temp [MessageHandshakeSize]byte
	var elem QueueHandshakeElement

	for {
		select {
		case elem = <-device.queue.handshake:
		case <-device.signal.stop:
			return
		}

		// handle cookie fields and ratelimiting
		switch elem.msgType {
		case MessageCookieReplyType:

			// unmarshal packet
			logger.Wlog.SaveDebugLog("Process cookie reply from:" + elem.source.String())

			var reply MessageCookieReply
			reader := bytes.NewReader(elem.packet)
			err := binary.Read(reader, binary.LittleEndian, &reply)
			if err != nil {
				logger.Wlog.SaveDebugLog("Failed to decode cookie reply")
				return
			}

			// lookup peer and consume response
			entry := device.indices.Lookup(reply.Receiver)
			if entry.peer == nil {
				return
			}
			entry.peer.mac.ConsumeReply(&reply)
			continue
		case MessageInitiationType, MessageResponseType:

			// check mac fields and ratelimit
			if !device.mac.CheckMAC1(elem.packet) {
				logger.Wlog.SaveDebugLog("Received packet with invalid mac1")
				return
			}

			if device.IsUnderLoad() {
				if !device.mac.CheckMAC2(elem.packet, elem.source) {
					// construct cookie reply
					logger.Wlog.SaveDebugLog("Sending cookie reply to:" + elem.source.String())

					sender := binary.LittleEndian.Uint32(elem.packet[4:8]) // "sender" always follows "type"
					reply, err := device.mac.CreateReply(elem.packet, sender, elem.source)
					if err != nil {
						logger.Wlog.SaveDebugLog("Failed to create cookie reply:" + err.Error())
						return
					}

					// marshal and send reply
					writer := bytes.NewBuffer(temp[:0])
					binary.Write(writer, binary.LittleEndian, reply)
					_, err = device.net.conn.WriteToUDP(
						writer.Bytes(),
						elem.source,
					)
					if err != nil {
						time.Sleep(2 * time.Second)
						changeNetwork(device, Endpoint)

						logger.Wlog.SaveDebugLog("Failed to send cookie reply:" + err.Error())
					}
					continue
				}

				if !device.ratelimiter.Allow(elem.source.IP) {
					continue
				}
			}

		default:
			logger.Wlog.SaveDebugLog("Invalid packet ended up in the handshake queue")
			continue
		}

		// handle handshake initation/response content
		switch elem.msgType {
		case MessageInitiationType:
			// unmarshal
			var msg MessageInitiation
			reader := bytes.NewReader(elem.packet)
			err := binary.Read(reader, binary.LittleEndian, &msg)
			if err != nil {
				logger.Wlog.SaveErrLog("Failed to decode initiation message")
				continue
			}

			// consume initiation
			peer := device.ConsumeMessageInitiation(&msg)
			if peer == nil {
				logger.Wlog.SaveInfoLog("Recieved invalid initiation message from:" + elem.source.IP.String() + strconv.Itoa(elem.source.Port))
				continue
			}

			// update timers
			peer.TimerAnyAuthenticatedPacketTraversal()
			peer.TimerAnyAuthenticatedPacketReceived()

			// update endpoint
			// TODO: Discover destination address also, only update on change
			peer.mutex.Lock()
			peer.endpoint = elem.source
			peer.mutex.Unlock()

			// create response
			response, err := device.CreateMessageResponse(peer)
			if err != nil {
				logger.Wlog.SaveErrLog("Failed to create response message:" + err.Error())
				continue
			}

			peer.TimerEphemeralKeyCreated()
			peer.NewKeyPair()

			logger.Wlog.SaveDebugLog("Creating response message for: " + peer.String())

			writer := bytes.NewBuffer(temp[:0])
			binary.Write(writer, binary.LittleEndian, response)
			packet := writer.Bytes()
			peer.mac.AddMacs(packet)

			// send response
			_, err = peer.SendBuffer(packet)
			if err == nil {
				peer.TimerAnyAuthenticatedPacketTraversal()
			} else {
				logger.Wlog.SaveInfoLog("RoutineHandshake发送失败")
			}

		case MessageResponseType:
			// unmarshal
			var msg MessageResponse
			reader := bytes.NewReader(elem.packet)
			err := binary.Read(reader, binary.LittleEndian, &msg)
			if err != nil {
				logger.Wlog.SaveErrLog("Failed to decode response message")
				continue
			}

			// consume response
			peer := device.ConsumeMessageResponse(&msg)
			if peer == nil {
				logger.Wlog.SaveInfoLog("Recieved invalid response message from " + elem.source.IP.String() + strconv.Itoa(elem.source.Port))
				continue
			}

			if initiationNum%60 == 0 {
				logger.Wlog.SaveDebugLog("Received handshake initation,5s,num:" + strconv.FormatInt(initiationNum, 10))
			}
			initiationNum++

			peer.TimerEphemeralKeyCreated()

			// update timers
			peer.TimerAnyAuthenticatedPacketTraversal()
			peer.TimerAnyAuthenticatedPacketReceived()
			peer.TimerHandshakeComplete()

			// derive key-pair
			peer.NewKeyPair()
			peer.SendKeepAlive()
		}
	}
}

func (peer *Peer) RoutineSequentialReceiver() {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover RoutineSequentialReceiver err:", err))
		}
	}()

	device := peer.device
	for {

		select {
		case <-peer.signal.stop:
			return
		case elem := <-peer.queue.inbound:
			// wait for decryption
			elem.mutex.Lock()
			if elem.IsDropped() {
				continue
			}

			// check for replay
			if !elem.keyPair.replayFilter.ValidateCounter(elem.counter) {
				continue
			}

			peer.TimerAnyAuthenticatedPacketTraversal()
			peer.TimerAnyAuthenticatedPacketReceived()
			peer.KeepKeyFreshReceiving()

			// check if using new key-pair

			kp := &peer.keyPairs
			kp.mutex.Lock()
			if kp.next == elem.keyPair {
				peer.TimerHandshakeComplete()
				if kp.previous != nil {
					device.DeleteKeyPair(kp.previous)
				}
				kp.previous = kp.current
				kp.current = kp.next
				kp.next = nil
			}
			kp.mutex.Unlock()

			// check for keep-alive

			if len(elem.packet) == 0 {
				if keepAliveNum == 0 || keepAliveNum%300 == 0 {
					logger.Wlog.SaveDebugLog("Received keep-alive,15s,num:" + strconv.FormatInt(keepAliveNum, 10))
				}
				keepAliveNum++
				continue
			}
			peer.TimerDataReceived()

			// verify source and strip padding

			switch elem.packet[0] >> 4 {
			case ipv4.Version:

				// strip padding

				if len(elem.packet) < ipv4.HeaderLen {
					continue
				}

				field := elem.packet[IPv4offsetTotalLength : IPv4offsetTotalLength+2]
				length := binary.BigEndian.Uint16(field)
				if int(length) > len(elem.packet) || int(length) < ipv4.HeaderLen {
					continue
				}

				elem.packet = elem.packet[:length]

				// verify IPv4 source

				src := elem.packet[IPv4offsetSrc : IPv4offsetSrc+net.IPv4len]
				if device.routingTable.LookupIPv4(src) != peer {
					logger.Wlog.SaveInfoLog("Packet with unallowed source IP from " + peer.String())
					continue
				}

			case ipv6.Version:

				// strip padding
				if len(elem.packet) < ipv6.HeaderLen {
					continue
				}

				field := elem.packet[IPv6offsetPayloadLength : IPv6offsetPayloadLength+2]
				length := binary.BigEndian.Uint16(field)
				length += ipv6.HeaderLen
				if int(length) > len(elem.packet) {
					continue
				}

				elem.packet = elem.packet[:length]

				// verify IPv6 source

				src := elem.packet[IPv6offsetSrc : IPv6offsetSrc+net.IPv6len]
				if device.routingTable.LookupIPv6(src) != peer {
					logger.Wlog.SaveInfoLog("Packet with unallowed source IP from " + peer.String())
					continue
				}

			default:
				logger.Wlog.SaveInfoLog("Packet with invalid IP version from " + peer.String())
				continue
			}

			_, err := device.tun.device.Write(elem.packet)
			device.PutMessageBuffer(elem.buffer)
			if err != nil {
				logger.Wlog.SaveErrLog("Failed to write packet to TUN device:" + err.Error())
			}
			//else {
			//	logger.Wlog.SaveInfoLog("Success to TUN device")
			//}
		}
	}
}
