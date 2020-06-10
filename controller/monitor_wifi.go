package controller

import (
	"bt/logger"

	"fmt"
	"net"
	"strings"
	"time"
)

func monitorWifi(device *Device) {
	if !IsiOS {
		return
	}
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if strings.HasPrefix(device.net.conn.LocalAddr().String(), "192.") {
				continue
			}

			interfaces, err := net.Interfaces()
			if err != nil {
				logger.Wlog.SaveInfoLog("获取网卡失败:" + err.Error())
				continue
			}

			for _, v := range interfaces {
				if v.Name == "en0" {
					addresses, _ := v.Addrs()
					for _, v := range addresses {
						if v.String() != "" && strings.HasPrefix(v.String(), "192.") {
							interAddr := strings.Split(v.String(), "/")
							localAddr := strings.Split(device.net.conn.LocalAddr().String(), ":")
							if interAddr[0] != localAddr[0] {
								logger.Wlog.SaveInfoLog(fmt.Sprintf("监听网络变化:旧:%s,新:%s", localAddr[0], interAddr[0]))
								changeNetwork(device, Endpoint)
							}
						}
					}
				}

			}
		}
	}
}
