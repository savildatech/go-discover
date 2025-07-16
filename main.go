package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"syscall"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("PORT env var required")
		os.Exit(1)
	}

	subnetStr := os.Getenv("SUBNET")
	var broadcast string
	if subnetStr != "" {
		_, subnet, err := net.ParseCIDR(subnetStr)
		if err != nil {
			panic(err)
		}
		ip := subnet.IP.To4()
		if ip == nil {
			panic("IPv4 only")
		}
		mask := subnet.Mask
		broadcastIP := net.IPv4zero.To4()
		for i := 0; i < 4; i++ {
			broadcastIP[i] = ip[i] | ^mask[i]
		}
		broadcast = broadcastIP.String()
	} else {
		broadcast = "255.255.255.255"
	}

	// Listen
	laddr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		panic(err)
	}
	lconn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, from, err := lconn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			fmt.Printf("Received from %s: %s\n", from, string(buf[:n]))
		}
	}()

	// Send
	baddr, err := net.ResolveUDPAddr("udp", broadcast+":"+port)
	if err != nil {
		panic(err)
	}
	sconn, err := net.DialUDP("udp", nil, baddr)
	if err != nil {
		panic(err)
	}

	// Enable broadcast
	fd, err := sconn.File()
	if err != nil {
		panic(err)
	}
	err = syscall.SetsockoptInt(int(fd.Fd()), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	if err != nil {
		panic(err)
	}

	// Input loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		_, err := sconn.Write([]byte(msg))
		if err != nil {
			fmt.Println(err)
		}
	}
}