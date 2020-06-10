package bt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"bt/common"
	"bt/controller"
	"bt/logger"

	"golang.org/x/crypto/curve25519"
)

var (
	device *controller.Device
)

type Callback interface {
	CallFd(int)
	CallStatus(int)
	CallUploadFlow(int)
	CallDownloadFlow(int)
	CallDestinationIP(string)
}

type configData struct {
	OwnPrivate   string `json:"own_private"`
	OwnPublic    string `json:"own_public"`
	TheirPublic  string `json:"their_public"`
	Endpoint     string `json:"endpoint"`
	AllowIp      string `json:"allow_ip"`
	LogPath      string `json:"log_path"`
	IsIOS        string `json:"is_iOS"`
	Ts           uint32 `json:"ts"`
	Sign         string `json:"sign"`
	Netmask      uint32 `json:"netmask"`
	IntervalTime int64  `json:"interval_time"`
	IsCallbackIp int    `json:"is_callback_ip"`
}

func Init(fd int, jsonFomt string) string {
	if fd <= 0 {
		return "tun fd有误:" + strconv.Itoa(fd)
	}
	var values configData
	err := json.Unmarshal([]byte(jsonFomt), &values)
	if err != nil {
		return err.Error()
	}

	//init logger
	err = logger.LogDiaryInit(values.LogPath)
	if err != nil {
		return err.Error()
	}

	logger.Wlog.SaveInfoLog("init start，config info:" + jsonFomt)

	if values.IsIOS == "1" {
		controller.IsiOS = true
	}

	controller.Ts = values.Ts
	controller.Sign = values.Sign
	controller.AllowIp = values.AllowIp
	controller.Netmask = values.Netmask
	controller.Endpoint = values.Endpoint
	if values.IntervalTime > 0 {
		controller.IntervalTime = values.IntervalTime
	}

	debug.SetGCPercent(10)

	// open TUN device
	tun, err := controller.CreateTUN(fd)
	if err != nil {
		logger.Wlog.SaveInfoLog(err.Error())
		return err.Error()
	}

	// create controller device
	device = controller.NewDevice(tun)
	config := make([]string, 6)
	config[0] = "own_private=" + values.OwnPrivate
	config[1] = "own_public=" + values.OwnPublic
	config[2] = "their_public=" + values.TheirPublic
	config[3] = "endpoint=" + values.Endpoint
	config[4] = "allowed_ip=0.0.0.0/0"
	config[5] = "persistent_keepalive_interval=15"

	errMsg := controller.SetOperation(device, config)
	if errMsg != "" {
		logger.Wlog.SaveInfoLog(errMsg)
		return errMsg
	}

	logger.Wlog.SaveInfoLog(fmt.Sprintf("Version:%s", time.Now().Format("2006-01-02")))
	logger.Wlog.SaveInfoLog("init finish")

	return ""
}

func Start(rfd int32, cb Callback) string {
	logger.Wlog.SaveInfoLog("enter start...")

	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("程序结束:", err))
		}
	}()

	if rfd < 0 {
		msg := "pipe fd is error:" + strconv.Itoa(int(rfd))
		logger.Wlog.SaveInfoLog(msg)
		return msg
	}

	logger.Wlog.SaveInfoLog("starting success...")
	go device.StartConnection()
	go status(cb)

	p := make([]byte, 1)
	syscall.Read(int(rfd), p)

	device.Close()

	// clean up UAPI bind
	logger.Wlog.SaveInfoLog("Closing")
	return ""
}

func status(c Callback) {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover status err:", err))
		}
	}()
	for {
		select {
		case <-device.WaitChannel():
			return
		//case n := <-controller.UdpfdChan:
		//	c.CallUDPFd(n)
		case n := <-controller.StatusChan:
			c.CallStatus(n)
		case fd := <-controller.FdChan:
			c.CallFd(fd)
			//case n := <-controller.UploadFlowChan:
			//	c.CallUploadFlow(n)
			//case n := <-controller.DownloadFlowChan:
			//	c.CallDownloadFlow(n)
			//case str := <-controller.DestinationIPChan:
			//	c.CallDestinationIP(str)
		}
	}
}

func GetPriAndPubKey() string {
	random := rand.Reader

	var pri, pub [32]byte
	_, err := io.ReadFull(random, pri[:])
	if err != nil {
		return ""
	}

	pri[0] &= 248
	pri[31] &= 127
	pri[31] |= 64

	curve25519.ScalarBaseMult(&pub, &pri)

	private := base64.StdEncoding.EncodeToString(pri[:])
	public := base64.StdEncoding.EncodeToString(pub[:])

	return private + "," + public
}

func GetDomain(domain, secret, isAbroad string) string {
	var err error
	var qtMsg []byte
	msgChan := make(chan []byte, 3)
	dnsName := []string{"114.114.114.114:53", "223.5.5.5:53", "119.29.29.29:53", "180.76.76.76:53"}
	if isAbroad == "1" {
		dnsName = []string{"8.8.8.8:53"}
	}

	for _, v := range dnsName {
		go func() {
			qtMsg, err = controller.SendDNSReq(v, domain, msgChan)
			if err != nil {
				fmt.Println("failed to req:", err)
			}
		}()
	}

L:
	for {
		select {
		case b := <-msgChan:
			qtMsg = b
			break L
		case <-time.After(6 * time.Second):
			break L
		}
	}

	if len(qtMsg) == 0 {
		return ""
	}

	if secret == "" {
		secret = "wriQVbeJqBWC2qJ0"
	}

	aesDec := common.AesEncrypt{}
	v, err := aesDec.Decrypt(qtMsg, []byte(secret))
	if err != nil {
		fmt.Println("failed to aes decrypt:", err)
		return ""
	}
	return string(v)
}
