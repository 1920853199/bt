package controller

//
//import (
//	"crypto/md5"
//	"encoding/hex"
//	"io/ioutil"
//	"net/http"
//	"strconv"
//	"strings"
//	"time"
//)
//
//const (
//	day = 15
//	key = "zcxvh14ikjsd901saf"
//)
//
//type Domain struct{}
//
//func (d *Domain) md5Hash(text string) string {
//	hasher := md5.New()
//	hasher.Write([]byte(text))
//	return hex.EncodeToString(hasher.Sum(nil))
//}
//
//func (d *Domain) GetDomain(first string) string {
//	com := strings.Split(first, ",")
//	if len(com) < 1 {
//		return ""
//	}
//	now := time.Now().AddDate(0, 0, -10)
//	nodeTime := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.Local)
//	var startTime, endTime time.Time
//
//	if now.Day() < day {
//		startTime = nodeTime.AddDate(0, -1, 0)
//		endTime = nodeTime
//	} else {
//		startTime = nodeTime
//		endTime = nodeTime.AddDate(0, 1, 0)
//	}
//
//	for startTime.Before(endTime) {
//		k := strconv.FormatInt(startTime.Unix(), 10)
//		if startTime.Day()%2 == 0 {
//			k += key
//		} else {
//			k = key + k
//		}
//
//		domain := d.md5Hash(k)
//		domain = domain[8:24]
//		url := d.verifyDomain(startTime.Format("2006-01-02"), domain, com)
//		if url != "" {
//			return url
//		}
//
//		startTime = startTime.AddDate(0, 0, 1)
//	}
//
//	return ""
//}
//
//func (d *Domain) verifyDomain(nodeTime, domain string, first []string) string {
//	client := http.Client{Timeout: 2 * time.Second}
//	for _, v := range first {
//		host := "http://" + domain + "." + v
//		url := host + "/" + nodeTime + "/ping"
//
//		resp, err := client.Get(url)
//		if err != nil {
//			continue
//		}
//
//		b, err := ioutil.ReadAll(resp.Body)
//		resp.Body.Close()
//		if err != nil {
//			continue
//		}
//
//		if string(b) == nodeTime {
//			return host
//		}
//	}
//
//	return ""
//}
