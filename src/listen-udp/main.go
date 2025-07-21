package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":9999")
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

	fmt.Println("Listening for UDP broadcasts on :9999...")

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
