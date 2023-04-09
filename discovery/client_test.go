package discovery

import (
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"testing"
)

func Test_IpRange(t *testing.T) {

	// 计算内网IP范围
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("addrs failed, err=%s", err)
	}

	var (
		ip *net.IPNet
		ok bool
	)
	for i, addr := range addrs {
		ip, ok = addr.(*net.IPNet)
		if ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				it, _ := net.InterfaceByIndex(i)
				t.Logf("ip: %s, mask: %s, mac: %v", ip.IP, ip.Mask, it)
				//break
			}
		}
	}

	ipv4 := ip.IP.To4()
	var min, max uint32
	for i := 0; i < 4; i++ {
		b := uint32(ipv4[i] & ip.Mask[i])
		min += b << ((3 - uint(i)) * 8)
	}
	one, _ := ip.Mask.Size()
	max = min | uint32(math.Pow(2, float64(32-one))-1)

	t.Logf("ip range = %d --- %d", UInt32ToIP(min).To4(), UInt32ToIP(max).To4())

	var data []uint32
	for i := min; i < max; i++ {
		if i&0x000000ff == 0 {
			continue
		}
		data = append(data, i)
	}
	//t.Logf("ips = %v", data)
}

func UInt32ToIP(intIP uint32) net.IP {
	var bytes [4]byte
	bytes[0] = byte(intIP & 0xFF)
	bytes[1] = byte((intIP >> 8) & 0xFF)
	bytes[2] = byte((intIP >> 16) & 0xFF)
	bytes[3] = byte((intIP >> 24) & 0xFF)

	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0])
}

func Test_GetOutBoundIP(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		return
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, addr := range addrs {
		fmt.Println(addr)
	}
}

func GetOutBoundIP() (ip string, err error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		fmt.Println(err)
		return
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	fmt.Println(localAddr.String())
	ip = strings.Split(localAddr.String(), ":")[0]
	return
}
