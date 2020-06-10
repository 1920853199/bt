package controller

import (
	"crypto/cipher"
	"sync"
	"time"
)

type KeyPair struct {
	send         cipher.AEAD
	receive      cipher.AEAD
	replayFilter ReplayFilter
	sendNonce    uint64
	isInitiator  bool
	created      time.Time
	localIndex   uint32
	remoteIndex  uint32
}

type KeyPairs struct {
	mutex    sync.RWMutex
	current  *KeyPair
	previous *KeyPair
	next     *KeyPair // not yet "confirmed by transport"
}

func (kp *KeyPairs) Current() *KeyPair {
	kp.mutex.RLock()
	defer kp.mutex.RUnlock()
	return kp.current
}

func (device *Device) DeleteKeyPair(key *KeyPair) {
	device.indices.Delete(key.localIndex)
}
