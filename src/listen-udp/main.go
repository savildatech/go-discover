package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	subnet := os.Getenv("SUBNET")
	if subnet == "" {
		subnet = "0.0.0.0"
	}
	port := os.Getenv("UDP_PORT")
	if port == "" {
		port = "9999"
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", subnet, port))
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

	fmt.Printf("Listening for UDP broadcasts on %s:%s...\n", subnet, port)

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
