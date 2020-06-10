package controller

// import (
// 	"fmt"
// 	"net"
// 	"time"

// 	"bt/logger"
// )

// func Ping(host string) string {
// 	var seq int16 = 1
// 	const ECHO_REQUEST_HEAD_LEN = 8
// 	timeout := time.Duration(3 * time.Second)

// 	conn, err := net.DialTimeout("ip4:icmp", host, timeout)
// 	if err != nil {
// 		fmt.Println("Failed to dial ip4:icmp:" + err.Error())
// 		logger.SaveErrLog("Failed to dial ip4:icmp:" + err.Error())
// 		return ""
// 	}
// 	defer conn.Close()

// 	id0, id1 := genidentifier(host)

// 	sendN, recvN, lostN, shortT, longT, sumT := 0, 0, 0, -1, -1, 0

// 	for i := 5; i > 0; i-- {
// 		sendN++
// 		msg := make([]byte, ECHO_REQUEST_HEAD_LEN)
// 		msg[0] = 8
// 		msg[1] = 0
// 		msg[2] = 0
// 		msg[3] = 0
// 		msg[4], msg[5] = id0, id1
// 		msg[6], msg[7] = gensequence(seq)
// 		check := checkSum(msg)
// 		msg[2] = byte(check >> 8)
// 		msg[3] = byte(check & 255)

// 		conn, err = net.DialTimeout("ip:icmp", host, timeout)
// 		if err != nil {
// 			fmt.Println("Failed to dial icmp:" + err.Error())
// 			logger.SaveErrLog("Failed to dial icmp:" + err.Error())
// 			return ""
// 		}

// 		starttime := time.Now()
// 		conn.SetDeadline(starttime.Add(timeout))
// 		_, err = conn.Write(msg)

// 		const ECHO_REPLY_HEAD_LEN = 20
// 		receive := make([]byte, ECHO_REPLY_HEAD_LEN+ECHO_REQUEST_HEAD_LEN)
// 		_, err = conn.Read(receive)

// 		endduration := int(int64(time.Since(starttime)) / (1000 * 1000))
// 		sumT += endduration
// 		time.Sleep(1 * time.Second)
// 		if err != nil || receive[ECHO_REPLY_HEAD_LEN+4] != msg[4] || receive[ECHO_REPLY_HEAD_LEN+5] != msg[5] {
// 			lostN++
// 		} else {
// 			if shortT == -1 {
// 				shortT = endduration
// 			} else if shortT > endduration {
// 				shortT = endduration
// 			}
// 			if longT == -1 {
// 				longT = endduration
// 			} else if longT < endduration {
// 				longT = endduration
// 			}
// 			recvN++
// 		}
// 		seq++
// 	}

// 	t := sumT / sendN
// 	lost := (lostN * 100) / sendN

// 	return fmt.Sprintf("%d,%d", t, lost)
// }

// func checkSum(msg []byte) uint16 {
// 	sum := 0

// 	length := len(msg)
// 	for i := 0; i < length-1; i += 2 {
// 		sum += int(msg[i])*256 + int(msg[i+1])
// 	}
// 	if length%2 == 1 {
// 		sum += int(msg[length-1]) * 256 // notice here, why *256?
// 	}

// 	sum = (sum >> 16) + (sum & 0xffff)
// 	sum += (sum >> 16)
// 	var answer uint16 = uint16(^sum)
// 	return answer
// }

// func gensequence(v int16) (byte, byte) {
// 	ret1 := byte(v >> 8)
// 	ret2 := byte(v & 255)
// 	return ret1, ret2
// }

// func genidentifier(host string) (byte, byte) {
// 	return host[0], host[1]
// }
