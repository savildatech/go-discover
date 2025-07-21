package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	ifaceName := os.Getenv("INTERFACE")
	if ifaceName == "" {
		fmt.Println("INTERFACE env var required")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("PORT env var required")
		os.Exit(1)
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Printf("Error finding interface: %v\n", err)
		os.Exit(1)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		fmt.Printf("Error getting addresses: %v\n", err)
		os.Exit(1)
	}

	var ip net.IP
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			ip = ipNet.IP
			break
		}
	}
	if ip == nil {
		fmt.Println("No IPv4 address found on interface")
		os.Exit(1)
	}

	addr := &net.UDPAddr{
		IP:   ip,
		Port: atoi(port),
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error reading: %v\n", err)
			continue
		}
		fmt.Printf("Received %d bytes from %s: %s\n", n, remoteAddr, string(buf[:n]))
	}
}

func atoi(s string) int {
	i := 0
	for _, c := range s {
		i = i*10 + int(c-'0')
	}
	return i
}
