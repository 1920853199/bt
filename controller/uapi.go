package controller

import (
	"net"
	"strconv"
	"strings"
	"sync/atomic"
)

func SetOperation(device *Device, values []string) string {
	var peer *Peer

	for _, v := range values {
		// parse line
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return "Failed to unmarshal:" + v
		}
		key := parts[0]
		value := parts[1]

		switch key {
		case "own_private":
			var sk NoisePrivateKey
			err := sk.FromBase64(value)
			if err != nil {
				return "Failed to set own_private:" + err.Error()
			}

			device.privateKey = sk
		case "own_public":
			var pubKey NoisePublicKey
			err := pubKey.FromBase64(value)
			if err != nil {
				return "Failed to get peer by public_key:" + err.Error()
			}
			device.SetPublicKey(pubKey)
		case "their_public":
			var pubKey NoisePublicKey
			err := pubKey.FromBase64(value)
			if err != nil {
				return "Failed to get peer by public_key:" + err.Error()
			}

			// check if public key of peer equal to device
			// find peer referenced
			peer = device.peers
			if peer == nil {
				peer, err = device.NewPeer(pubKey)
				if err != nil {
					return "Failed to create new peer:" + err.Error()
				}
			}
		case "endpoint":
			addr, err := parseEndpoint(value)
			if err != nil {
				return "Failed to set endpoint:" + err.Error()
			}

			if peer == nil {
				peer = device.peers
			}

			peer.mutex.Lock()
			peer.endpoint = addr
			peer.mutex.Unlock()

			//create udp connection
			fd, err := createUDPConn(device, addr)
			if err != nil {
				return "Failed to set udp conn:" + err.Error()
			}

			go sendFd(fd)

			peer.device.net.conn = device.net.conn
		case "allowed_ip":
			_, network, err := net.ParseCIDR(value)
			if err != nil {
				return "Failed to set allowed_ip:" + err.Error()
			}

			ones, _ := network.Mask.Size()
			device.routingTable.Insert(network.IP, uint(ones), peer)
		case "persistent_keepalive_interval":
			// update keep-alive interval
			secs, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return "Failed to set persistent_keepalive_interval:" + err.Error()
			}

			old := atomic.SwapUint32(
				&peer.persistentKeepaliveInterval,
				uint32(secs),
			)

			// send immediate keep-alive

			if old == 0 && secs != 0 {
				if err != nil {
					return "Failed to get tun device status:" + err.Error()
				}
				peer.SendKeepAlive()
			}
		default:
			return "Invalid UAPI key (device configuration):" + v
		}
	}

	return ""
}
