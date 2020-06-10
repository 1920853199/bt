package controller

import (
	"encoding/binary"
	"encoding/hex"
	"net"
)

const (
	timestampSize = 4
	signSize      = 16
	allowIpSize   = 4
	netmaskSize   = 4
)

type timestampData [timestampSize]byte
type signData [signSize]byte

func newTimestampData() timestampData {
	var data timestampData
	binary.BigEndian.PutUint32(data[:], Ts)
	return data
}

func newSignData() signData {
	var data signData

	sign, _ := hex.DecodeString(Sign)

	copy(data[:], sign)
	return data
}

func newAllowIpData() [allowIpSize]byte {
	var data [allowIpSize]byte
	ip := net.ParseIP(AllowIp).To4()

	copy(data[:], ip[:])
	return data
}

func newNetmaskData() [netmaskSize]byte {
	var data [netmaskSize]byte
	binary.BigEndian.PutUint32(data[:], Netmask)
	return data
}
