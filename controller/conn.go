package controller

import (
	"errors"
	"net"

	"bt/logger"

	"golang.org/x/sys/unix"
)

func parseEndpoint(s string) (*net.UDPAddr, error) {
	// ensure that the host is an IP address
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip == nil {
		return nil, errors.New("Failed to parse IP address: " + host)
	}

	// parse address and port
	addr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		return nil, err
	}
	return addr, err
}

func createUDPConn(device *Device, raddr *net.UDPAddr) (int, error) {
	netc := &device.net
	netc.mutex.Lock()
	defer netc.mutex.Unlock()

	// close existing connection
	if netc.conn != nil {
		logger.Wlog.SaveInfoLog("断开旧的udp连接:" + netc.conn.LocalAddr().String())
		go netc.conn.Close()
		netc.conn = nil
	}

	// open new connection
	// listen on new address
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return -1, err
	}

	setMark(conn, uint32(fwmarkIoctl))
	netc.conn = conn

	logger.Wlog.SaveInfoLog("创建新的udp连接:" + conn.LocalAddr().String())

	// notify goroutines
	signalSend(device.signal.newUDPConn)
	f, err := conn.File()
	if err != nil {
		return -1, err
	}
	return int(f.Fd()), nil
}

var fwmarkIoctl int

func init() {
	if !IsiOS {
		fwmarkIoctl = 36
	}
}
func setMark(conn *net.UDPConn, mark uint32) error {
	if fwmarkIoctl == 0 {
		return nil
	}

	fd, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	err = fd.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, fwmarkIoctl, int(mark))
	})
	if err != nil {
		return err
	}

	return nil
}

func closeUDPConn(device *Device) {
	device.net.mutex.Lock()
	device.net.conn.Close()
	device.net.mutex.Unlock()
	signalSend(device.signal.newUDPConn)
}
