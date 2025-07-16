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
	baddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:"+port)
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