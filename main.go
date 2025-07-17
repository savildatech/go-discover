package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"crypto/rand"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	minBroadcastInterval = 30  // seconds
	maxBroadcastInterval = 60  // seconds
	cleanupInterval      = 30  // seconds
	peerTimeout          = 120 // seconds
)

type Peer struct {
	Hostname string    `json:"hostname"`
	IP       string    `json:"ip"`
	CPUUsage float64   `json:"cpu_usage"`
	MemUsage float64   `json:"mem_usage"`
	Service  string    `json:"service"`
	LastSeen time.Time `json:"last_seen"`
}

var (
	peers     = make(map[string]Peer)
	mu        sync.Mutex
	subnet    = os.Getenv("SUBNET") // e.g., "192.168.1.0/24"
	udpPort   = os.Getenv("UDP_PORT")
	httpPort  = os.Getenv("HTTP_PORT")
	service   = os.Getenv("SERVICE")
	localIP   string
	bcastAddr string
)

func main() {
	if udpPort == "" {
		udpPort = "9999"
	}
	if httpPort == "" {
		httpPort = "8080"
	}
	if subnet == "" {
		subnet = "192.168.1.0/24" // default
	}

	var err error
	localIP, err = getLocalIP(subnet)
	if err != nil {
		log.Fatal(err)
	}

	bcastAddr = getBroadcastAddr(subnet)

	go broadcastLoop()
	go listener()
	go cleanupLoop()

	http.HandleFunc("/servers", serversHandler)
	log.Fatal(http.ListenAndServe(":"+httpPort, nil))
}

func getLocalIP(subnet string) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
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
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ipnet.Contains(ip) && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no IP in subnet %s", subnet)
}

func getBroadcastAddr(subnet string) string {
	ip, ipnet, _ := net.ParseCIDR(subnet)
	bcast := make(net.IP, len(ip))
	copy(bcast, ip)
	for i := range bcast {
		bcast[i] |= ^ipnet.Mask[i]
	}
	return bcast.String() + ":" + udpPort
}

func broadcastLoop() {
	hostname, _ := os.Hostname()
	conn, err := net.Dial("udp", bcastAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		cpuUsage, _ := cpu.Percent(0, false)
		memUsage, _ := mem.VirtualMemory()
		data := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", hostname, localIP, cpuUsage[0], memUsage.UsedPercent, service)
		conn.Write([]byte(data))

		n, err := rand.Int(rand.Reader, big.NewInt(int64(maxBroadcastInterval-minBroadcastInterval+1)))
		if err != nil {
			n = big.NewInt(0) // fallback
		}
		sleep := minBroadcastInterval + int(n.Int64())
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

func listener() {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+udpPort)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if from.IP.String() == localIP {
			continue // ignore self
		}
		parts := strings.Split(string(buf[:n]), "|")
		if len(parts) != 5 {
			continue
		}
		hostname := parts[0]
		ip := parts[1]
		var cpu, mem float64
		fmt.Sscanf(parts[2], "%f", &cpu)
		fmt.Sscanf(parts[3], "%f", &mem)
		svc := parts[4]

		mu.Lock()
		peers[ip] = Peer{
			Hostname: hostname,
			IP:       ip,
			CPUUsage: cpu,
			MemUsage: mem,
			Service:  svc,
			LastSeen: time.Now(),
		}
		mu.Unlock()
	}
}

func cleanupLoop() {
	for {
		time.Sleep(time.Duration(cleanupInterval) * time.Second)
		now := time.Now()
		mu.Lock()
		for ip, p := range peers {
			if now.Sub(p.LastSeen) > time.Duration(peerTimeout)*time.Second {
				delete(peers, ip)
			}
		}
		mu.Unlock()
	}
}

func serversHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	list := make([]Peer, 0, len(peers))
	for _, p := range peers {
		list = append(list, p)
	}
	mu.Unlock()
	json.NewEncoder(w).Encode(list)
}
