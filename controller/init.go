package controller

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"bt/logger"
)

var (
	initiationNum     int64  = 0
	keepAliveNum      int64  = 0
	keepalivePassive         = 0
	firstConnSuccess         = false
	IsiOS                    = false
	Ts                uint32 = 0
	Sign                     = ""
	AllowIp                  = ""
	Endpoint                 = ""
	Netmask           uint32 = 0
	UploadFlowNum            = 0
	DownloadFlowNum          = 0
	IntervalTime      int64  = 50
	IntervalStartTime int64  = 0
	StatusChan        chan int
	UploadFlowChan    chan int
	DownloadFlowChan  chan int
	FdChan            chan int
	DestinationIPChan chan string
	keepaliveMutex    sync.Mutex
)

func init() {
	FdChan = make(chan int)
	DestinationIPChan = make(chan string, 10)
	StatusChan = make(chan int, 10)
	//DownloadFlowChan = make(chan int, 20)
	//UploadFlowChan = make(chan int, 20)
}

func checkIntervalTime(d *Device) {
	t := time.NewTicker(time.Second)

	for {
		select {
		case <-d.WaitChannel():
			t.Stop()
			return
		case <-t.C:
			now := time.Now().Unix()
			if now-IntervalStartTime > IntervalTime {
				logger.Wlog.SaveErrLog(fmt.Sprintf("当前时间:%d,重试时间:%d。未收到握手包应答", now, IntervalStartTime))
				IntervalStartTime = now
				go sendStatus(101)
			}
		}
	}

}

func sendUploadFlow() {
	if UploadFlowNum > 0 {
		UploadFlowChan <- UploadFlowNum
		UploadFlowNum = 0
	}
}
func sendDownloadFlow() {
	if DownloadFlowNum > 0 {
		DownloadFlowChan <- DownloadFlowNum
		DownloadFlowNum = 0
	}
}

func sendStatus(n int) {
	StatusChan <- n
}

func sendFd(fd int) {
	FdChan <- fd
}

//func sendDestinationIP(destinationIP string) {
//	//t := time.NewTicker(time.Second)
//	//
//	//for {
//	//	select {
//	//	case <-d.WaitChannel():
//	//		t.Stop()
//	//		return
//	//	case <-t.C:
//	//		if destinationIP == "" {
//	//			continue
//	//		}
//
//	DestinationIPChan <- destinationIP
//	//	}
//	//}
//}

//func taskSendFlow(d *Device) {
//	t := time.NewTicker(1 * time.Second)
//	for {
//		select {
//		case <-d.WaitChannel():
//			t.Stop()
//			return
//		case <-t.C:
//			sendUploadFlow()
//			sendDownloadFlow()
//		}
//	}
//}

func taskGC(d *Device) {
	if IsiOS {
		t := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-d.WaitChannel():
				t.Stop()
				return
			case <-t.C:
				debug.FreeOSMemory()
			}
		}
	}
}

func changeNetwork(device *Device, sourceAddr string) {
	result := SetOperation(device, []string{"endpoint=" + sourceAddr})
	if result != "" {
		logger.Wlog.SaveErrLog("网络切换出错：" + result)
	}
}
