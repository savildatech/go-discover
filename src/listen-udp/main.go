package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func findLocalIPInCIDR(cidrStr string) (string, error) {
	_, cidr, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return "", err
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
				continue
			}
			if cidr.Contains(ipnet.IP) {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no IP in CIDR %s", cidrStr)
}

func main() {
	cidr := os.Getenv("NETWORK")
	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		fmt.Println("Invalid PORT")
		os.Exit(1)
	}
	host, err := findLocalIPInCIDR(cidr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	listenAddr := net.JoinHostPort(host, portStr)
	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		fmt.Printf("Listen failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	buf := make([]byte, 65536)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			continue
		}
		fmt.Printf("From %s: %s\n", addr, string(buf[:n]))
	}
}
