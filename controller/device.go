package controller

import (
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"bt/logger"
)

type Device struct {
	tun struct {
		device *NativeTun
		mtu    int32
	}
	pool struct {
		messageBuffers sync.Pool
	}
	net struct {
		mutex sync.RWMutex
		conn  *net.UDPConn // UDP "connection"
	}
	// netFd int
	mutex        sync.RWMutex
	privateKey   NoisePrivateKey
	publicKey    NoisePublicKey
	routingTable RoutingTable
	indices      IndexTable
	queue        struct {
		encryption chan *QueueOutboundElement
		decryption chan *QueueInboundElement
		handshake  chan QueueHandshakeElement
	}
	signal struct {
		stop       chan struct{} // halts all go routines
		newUDPConn chan struct{} // a net.conn was set (consumed by the receiver routine)
	}
	underLoadUntil atomic.Value
	ratelimiter    Ratelimiter
	peers          *Peer
	mac            CookieChecker
}

/* Warning:
 * The caller must hold the device mutex (write lock)
 */
func removePeerUnsafe(device *Device) {
	peer := device.peers
	device.routingTable.RemovePeer(peer)
	device.peers = nil
	peer.Close()
}

func (device *Device) IsUnderLoad() bool {

	// check if currently under load

	now := time.Now()
	underLoad := len(device.queue.handshake) >= UnderLoadQueueSize
	if underLoad {
		device.underLoadUntil.Store(now.Add(time.Second))
		return true
	}

	// check if recently under load

	until := device.underLoadUntil.Load().(time.Time)
	return until.After(now)
}

func (device *Device) SetPublicKey(sk NoisePublicKey) error {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.publicKey = sk
	device.mac.Init(sk)

	// do DH precomputations

	rmKey := device.privateKey.IsZero()

	h := &device.peers.handshake
	h.mutex.Lock()
	if rmKey {
		h.precomputedStaticStatic = [NoisePublicKeySize]byte{}
	} else {
		h.precomputedStaticStatic = device.privateKey.sharedSecret(h.remoteStatic)
		if isZero(h.precomputedStaticStatic[:]) {
			device.routingTable.RemovePeer(device.peers)
			device.peers = nil
		}
	}
	h.mutex.Unlock()

	return nil
}

func (device *Device) GetMessageBuffer() *[MaxMessageSize]byte {
	return device.pool.messageBuffers.Get().(*[MaxMessageSize]byte)
}

func (device *Device) PutMessageBuffer(msg *[MaxMessageSize]byte) {
	device.pool.messageBuffers.Put(msg)
}

func NewDevice(tun *NativeTun) *Device {
	device := new(Device)

	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.peers = &Peer{}
	device.tun.device = tun
	device.indices.Init()
	device.ratelimiter.Init()
	device.routingTable.Reset()
	device.underLoadUntil.Store(time.Time{})

	// setup pools
	device.pool.messageBuffers = sync.Pool{
		New: func() interface{} {
			return new([MaxMessageSize]byte)
		},
	}

	// create queues

	device.queue.handshake = make(chan QueueHandshakeElement, QueueHandshakeSize)
	device.queue.encryption = make(chan *QueueOutboundElement, QueueOutboundSize)
	device.queue.decryption = make(chan *QueueInboundElement, QueueInboundSize)

	// prepare signals

	device.signal.stop = make(chan struct{})
	device.signal.newUDPConn = make(chan struct{}, 1)

	return device
}

func (d *Device) StartConnection() {
	IntervalStartTime = time.Now().Unix()
	go checkIntervalTime(d)
	go monitorWifi(d)
	go d.peers.RoutineNonce()
	go d.peers.RoutineTimerHandler()
	go d.peers.RoutineHandshakeInitiator()
	go d.peers.RoutineSequentialSender()
	go d.peers.RoutineSequentialReceiver()

	time.Sleep(time.Second / 2)
	signalSend(d.peers.signal.handshakeReset)
	//go taskSendFlow(d)
	go taskGC(d)

	// start workers
	go d.RoutineEncryption()
	go d.RoutineDecryption()
	go d.RoutineHandshake()

	go d.ratelimiter.RoutineGarbageCollector(d.signal.stop)
	go d.RoutineReadFromTUN()
	go d.RoutineReceiveIncomming()
}

func (device *Device) LookupPeer() *Peer {
	device.mutex.RLock()
	defer device.mutex.RUnlock()
	return device.peers
}

func (device *Device) RemovePeer() {
	device.mutex.Lock()
	defer device.mutex.Unlock()
	removePeerUnsafe(device)
}

func (device *Device) Close() {
	syscall.Close(device.tun.device.fd)
	device.RemovePeer()
	close(device.signal.stop)
	closeUDPConn(device)
	device.tun.device.Close()
	logger.Wlog.Close()
}

func (device *Device) WaitChannel() chan struct{} {
	return device.signal.stop
}
