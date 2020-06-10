package controller

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	NoisePublicKeySize  = 32
	NoisePrivateKeySize = 32
)

type (
	NoisePublicKey    [NoisePublicKeySize]byte
	NoisePrivateKey   [NoisePrivateKeySize]byte
	NoiseSymmetricKey [chacha20poly1305.KeySize]byte
	NoiseNonce        uint64 // padded to 12-bytes
)

func loadExactBase64(dst []byte, src string) error {
	slice, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return err
	}
	if len(slice) != len(dst) {
		return errors.New("Hex string does not fit the slice")
	}
	copy(dst, slice)
	return nil
}

func (key NoisePrivateKey) IsZero() bool {
	var zero NoisePrivateKey
	return key.Equals(zero)
}

func (key NoisePrivateKey) Equals(tar NoisePrivateKey) bool {
	return subtle.ConstantTimeCompare(key[:], tar[:]) == 1
}

func (key *NoisePrivateKey) FromBase64(src string) error {
	return loadExactBase64(key[:], src)
}

func (key NoisePrivateKey) ToHex() string {
	return hex.EncodeToString(key[:])
}

func (key *NoisePublicKey) FromBase64(src string) error {
	return loadExactBase64(key[:], src)
}

func (key NoisePublicKey) ToHex() string {
	return hex.EncodeToString(key[:])
}

func (key NoisePublicKey) IsZero() bool {
	var zero NoisePublicKey
	return key.Equals(zero)
}

func (key NoisePublicKey) Equals(tar NoisePublicKey) bool {
	return subtle.ConstantTimeCompare(key[:], tar[:]) == 1
}

func (key *NoiseSymmetricKey) FromBase64(src string) error {
	return loadExactBase64(key[:], src)
}

func (key NoiseSymmetricKey) ToHex() string {
	return hex.EncodeToString(key[:])
}
