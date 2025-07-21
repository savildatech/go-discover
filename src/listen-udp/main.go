package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func findLocalIPInNetwork(network *net.IPNet) net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip.To4() == nil {
				continue // Skip IPv6
			}
			if network.Contains(ip) {
				return ip
			}
		}
	}
	return nil
}

func main() {
	subnet := os.Getenv("SUBNET")
	if subnet == "" {
		subnet = "0.0.0.0"
	}
	port := os.Getenv("UDP_PORT")
	if port == "" {
		port = "9999"
	}

	var bindIP string
	if subnet == "0.0.0.0" {
		bindIP = "0.0.0.0"
	} else {
		cidr := subnet
		if !strings.Contains(cidr, "/") {
			cidr += "/24"
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing CIDR: %v\n", err)
			os.Exit(1)
		}
		ip := findLocalIPInNetwork(network)
		if ip == nil {
			fmt.Fprintf(os.Stderr, "No local IP found in subnet %s\n", cidr)
			os.Exit(1)
		}
		bindIP = ip.String()
	}

	addr, err := net.ResolveUDPAddr("udp", bindIP+":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Listening for UDP broadcasts on %s:%s...\n", bindIP, port)

	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading: %v\n", err)
			continue
		}
		fmt.Printf("Received from %s: %s\n", remoteAddr.String(), string(buffer[:n]))
	}
}
