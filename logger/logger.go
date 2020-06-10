package logger

import (
	"net"
	"os"
	"sync"
	"time"
)

var Wlog *Wlogger

type Wlogger struct {
	fd   *os.File
	conn *net.UDPConn
	sync.Mutex
}

func LogDiaryInit(path string) error {
	if path == "" {
		//创建err_diary.log文件
		os.Mkdir("diary", os.ModePerm)
		path = "./diary/info_diary.log"
	}

	fd, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	Wlog = new(Wlogger)
	Wlog.fd = fd

	return nil
}

func (w *Wlogger) Close() {
	w.Lock()
	defer w.Unlock()
	w.fd.Close()
	w = nil
}

func (w *Wlogger) SaveInfoLog(msg string) {
	if Wlog != nil {
		now := time.Now().Local().Format("2006-01-02 15:04:05")
		wmsg := now + " [I] " + msg + "\n"

		w.Lock()
		defer w.Unlock()

		w.fd.WriteString(wmsg)
		w.fd.Sync()
	}
}

func (w *Wlogger) SaveDebugLog(msg string) {
	if Wlog != nil {
		now := time.Now().Local().Format("2006-01-02 15:04:05")
		wmsg := now + " [I] " + msg + "\n"

		w.Lock()
		defer w.Unlock()

		w.fd.WriteString(wmsg)
		w.fd.Sync()
	}
}

func (w *Wlogger) SaveErrLog(msg string) {
	if Wlog != nil {
		now := time.Now().Local().Format("2006-01-02 15:04:05")
		wmsg := now + " [I] " + msg + "\n"

		w.Lock()
		defer w.Unlock()

		w.fd.WriteString(wmsg)
		w.fd.Sync()
	}
}
