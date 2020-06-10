package controller

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

type dnsHeader struct {
	Id                                 uint16
	Bits                               uint16
	Qdcount, Ancount, Nscount, Arcount uint16
}

func (header *dnsHeader) SetFlag(QR uint16, OperationCode uint16, AuthoritativeAnswer uint16, Truncation uint16, RecursionDesired uint16, RecursionAvailable uint16, ResponseCode uint16) {
	header.Bits = QR<<15 + OperationCode<<11 + AuthoritativeAnswer<<10 + Truncation<<9 + RecursionDesired<<8 + RecursionAvailable<<7 + ResponseCode
}

type dnsQuery struct {
	QuestionType  uint16
	QuestionClass uint16
}

func parseDomainName(domain string) []byte {
	var buffer bytes.Buffer
	segments := strings.Split(domain, ".")

	for _, v := range segments {
		binary.Write(&buffer, binary.BigEndian, byte(len(v)))
		binary.Write(&buffer, binary.BigEndian, []byte(v))
	}
	binary.Write(&buffer, binary.BigEndian, byte(0x00))

	return buffer.Bytes()
}

func SendDNSReq(dnsServer, domain string, msgChan chan []byte) ([]byte, error) {
	var buffer bytes.Buffer

	requestHeader := dnsHeader{
		Id:      0x0010,
		Qdcount: 1,
		Ancount: 0,
		Nscount: 0,
		Arcount: 0,
	}
	requestHeader.SetFlag(0, 0, 0, 0, 1, 0, 0)

	requestQuery := dnsQuery{
		QuestionType:  16,
		QuestionClass: 1,
	}

	conn, err := net.Dial("udp", dnsServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	binary.Write(&buffer, binary.BigEndian, requestHeader)
	binary.Write(&buffer, binary.BigEndian, parseDomainName(domain))
	binary.Write(&buffer, binary.BigEndian, requestQuery)

	msg := buffer.Bytes()

	buf := make([]byte, 1024)
	if _, err := conn.Write(msg); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	length, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	if length <= 13 || len(msg)+13 > length {
		return nil, errors.New("长度有误:" + strconv.Itoa(length))
	}

	resultMsg := buf[len(msg)+13 : length]
	msgChan <- resultMsg
	return resultMsg, nil
}
