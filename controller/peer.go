package controller

import (
	"net"
	"sync"
	"time"
)

type Peer struct {
	mutex                       sync.RWMutex
	persistentKeepaliveInterval uint32
	keyPairs                    KeyPairs
	handshake                   Handshake
	device                      *Device
	endpoint                    *net.UDPAddr
	time                        struct {
		mutex         sync.RWMutex
		lastSend      time.Time // last send message
		lastHandshake time.Time // last completed handshake
		nextKeepalive time.Time
	}
	signal struct {
		newKeyPair         chan struct{} // (size 1) : a new key pair was generated
		handshakeBegin     chan struct{} // (size 1) : request that a new handshake be started ("queue handshake")
		handshakeCompleted chan struct{} // (size 1) : handshake completed
		handshakeReset     chan struct{} // (size 1) : reset handshake negotiation state
		flushNonceQueue    chan struct{} // (size 1) : empty queued packets
		messageSend        chan struct{} // (size 1) : a message was send to the peer
		messageReceived    chan struct{} // (size 1) : an authenticated message was received
		stop               chan struct{} // (size 0) : close to stop all goroutines for peer
	}
	timer struct {
		// state related to bt timers
		keepalivePersistent *time.Timer // set for persistent keepalives
		keepalivePassive    *time.Timer // set upon recieving messages
		newHandshake        *time.Timer // begin a new handshake (after Keepalive + RekeyTimeout)
		zeroAllKeys         *time.Timer // zero all key material (after RejectAfterTime*3)
		handshakeDeadline   *time.Timer // Current handshake must be completed

		pendingKeepalivePassive bool
		pendingNewHandshake     bool
		pendingZeroAllKeys      bool

		needAnotherKeepalive    bool
		sendLastMinuteHandshake bool
	}
	queue struct {
		nonce    chan *QueueOutboundElement // nonce / pre-handshake queue
		outbound chan *QueueOutboundElement // sequential ordering of work
		inbound  chan *QueueInboundElement  // sequential ordering of work
	}
	mac CookieGenerator
}

func (device *Device) NewPeer(pk NoisePublicKey) (*Peer, error) {
	// create peer
	peer := new(Peer)
	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	peer.mac.Init(pk)
	peer.device = device

	peer.timer.keepalivePersistent = NewStoppedTimer()
	peer.timer.keepalivePassive = NewStoppedTimer()
	peer.timer.newHandshake = NewStoppedTimer()
	peer.timer.zeroAllKeys = NewStoppedTimer()

	// assign id for debugging
	device.peers = peer

	// precompute DH

	handshake := &peer.handshake
	handshake.mutex.Lock()
	handshake.remoteStatic = pk
	handshake.precomputedStaticStatic = device.privateKey.sharedSecret(handshake.remoteStatic)
	handshake.mutex.Unlock()

	// prepare queuing

	peer.queue.nonce = make(chan *QueueOutboundElement, QueueOutboundSize)
	peer.queue.outbound = make(chan *QueueOutboundElement, QueueOutboundSize)
	peer.queue.inbound = make(chan *QueueInboundElement, QueueInboundSize)

	// prepare signaling & routines

	peer.signal.stop = make(chan struct{})
	peer.signal.newKeyPair = make(chan struct{}, 1)
	peer.signal.handshakeBegin = make(chan struct{}, 1)
	peer.signal.handshakeReset = make(chan struct{}, 1)
	peer.signal.handshakeCompleted = make(chan struct{}, 1)
	peer.signal.flushNonceQueue = make(chan struct{}, 1)

	return peer, nil
}

func (peer *Peer) String() string {
	str := "null"
	conn := peer.device.net.conn
	if conn != nil {
		str = conn.LocalAddr().String()
	}
	return "peer(" + str + ")"
}

func (peer *Peer) Close() {
	//stop timer
	peer.timer.keepalivePersistent.Stop()
	peer.timer.keepalivePassive.Stop()
	peer.timer.zeroAllKeys.Stop()
	peer.timer.newHandshake.Stop()
	peer.timer.handshakeDeadline.Stop()

	//close queues
	close(peer.signal.stop)
	close(peer.queue.nonce)
	close(peer.queue.outbound)
	close(peer.queue.inbound)

	close(peer.signal.newKeyPair)
	close(peer.signal.handshakeBegin)
	close(peer.signal.flushNonceQueue)
	close(peer.signal.handshakeCompleted)

	//clear key pairs
	device := peer.device
	kp := &peer.keyPairs
	kp.mutex.Lock()
	device.DeleteKeyPair(kp.previous)
	device.DeleteKeyPair(kp.current)
	device.DeleteKeyPair(kp.next)
	kp.previous = nil
	kp.current = nil
	kp.next = nil
	kp.mutex.Unlock()

	//clear handshake state
	hs := &peer.handshake
	hs.mutex.Lock()
	device.indices.Delete(hs.localIndex)
	hs.Clear()
	hs.mutex.Unlock()

}
