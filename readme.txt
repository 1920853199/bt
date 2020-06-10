##编译android

gomobile bind -target=android

##编译iOS

gomobile bind -target=ios


##查看网卡名称：

adb shell route

##抓 wlan0 网卡：

adb shell tcpdump -i wlan0 -p -s 0 -vv -w /sdcard/a1.pcap

adb pull /sdcard/a1.pcap .


##抓 tun0 网卡：

adb shell tcpdump -i tun0 -p -s 0 -vv -w /sdcard/a2.pcap

adb pull /sdcard/a2.pcap .