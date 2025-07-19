package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	netinfo "github.com/shirou/gopsutil/v4/net"
)

type Resources struct {
	Hostname    string  `json:"hostname"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	TxKbps      float64 `json:"tx_kbps"`
	RxKbps      float64 `json:"rx_kbps"`
	ActiveConns float64 `json:"active_conns"`
	Custom      string  `json:"custom"`
}

func main() {
	broadcastAddr := os.Getenv("BROADCAST_ADDR")
	if broadcastAddr == "" {
		log.Fatal("Missing BROADCAST_ADDR")
	}
	portStr := os.Getenv("PORT")
	if portStr == "" {
		log.Fatal("Missing PORT")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		log.Fatal("Invalid PORT")
	}
	minStr := os.Getenv("MIN_INTERVAL")
	if minStr == "" {
		log.Fatal("Missing MIN_INTERVAL")
	}
	min, err := strconv.Atoi(minStr)
	if err != nil || min <= 0 {
		log.Fatal("Invalid MIN_INTERVAL")
	}
	maxStr := os.Getenv("MAX_INTERVAL")
	if maxStr == "" {
		log.Fatal("Missing MAX_INTERVAL")
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil || max <= 0 {
		log.Fatal("Invalid MAX_INTERVAL")
	}
	if min >= max {
		log.Fatal("MIN_INTERVAL must be less than MAX_INTERVAL")
	}
	avgStr := os.Getenv("AVG_SECONDS")
	if avgStr == "" {
		log.Fatal("Missing AVG_SECONDS")
	}
	avgSec, err := strconv.Atoi(avgStr)
	if err != nil || avgSec <= 0 {
		log.Fatal("Invalid AVG_SECONDS")
	}
	sampleStr := os.Getenv("SAMPLE_INTERVAL")
	sampleInterval := 5
	if sampleStr != "" {
		sampleInterval, err = strconv.Atoi(sampleStr)
		if err != nil || sampleInterval <= 0 {
			log.Fatal("Invalid SAMPLE_INTERVAL")
		}
	}
	custom := os.Getenv("CUSTOM_STRING")
	if custom == "" {
		log.Fatal("Missing CUSTOM_STRING")
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("Hostname error:", err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", broadcastAddr+":"+portStr)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	maxSamples := avgSec / sampleInterval
	if maxSamples < 1 {
		maxSamples = 1
	}

	var mu sync.Mutex
	var cpuHist, memHist, txHist, rxHist, connHist []float64

	ctx, cancel := context.WithCancel(context.Background())
	go monitor(ctx, sampleInterval, maxSamples, &mu, &cpuHist, &memHist, &txHist, &rxHist, &connHist)

	// Initial sleep to collect samples
	time.Sleep(time.Duration(avgSec) * time.Second)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		os.Exit(0)
	}()

	for {
		bi, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
		if err != nil {
			log.Println("Random error:", err)
			continue
		}
		interval := min + int(bi.Int64())
		time.Sleep(time.Duration(interval) * time.Second)

		mu.Lock()
		cpuAvg := round(average(cpuHist), 2)
		memAvg := round(average(memHist), 2)
		txAvg := round(average(txHist), 2)
		rxAvg := round(average(rxHist), 2)
		connAvg := round(average(connHist), 2)
		mu.Unlock()

		if txAvg < 1 {
			txAvg = 0
		}
		if rxAvg < 1 {
			rxAvg = 0
		}

		data := Resources{
			Hostname:    hostname,
			CPU:         cpuAvg,
			Memory:      memAvg,
			TxKbps:      txAvg,
			RxKbps:      rxAvg,
			ActiveConns: connAvg,
			Custom:      custom,
		}
		js, err := json.Marshal(data)
		if err != nil {
			log.Println("JSON error:", err)
			continue
		}
		_, err = conn.Write(js)
		if err != nil {
			log.Println("Send error:", err)
		}
	}
}

func monitor(ctx context.Context, sampleInterval, maxSamples int, mu *sync.Mutex, cpuH, memH, txH, rxH, connH *[]float64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			prevTx, prevRx := getNetIO()

			percents, err := cpu.Percent(time.Duration(sampleInterval)*time.Second, false)
			var cpuP float64
			if err != nil {
				log.Println("CPU error:", err)
			} else if len(percents) > 0 {
				cpuP = percents[0]
			}

			currTx, currRx := getNetIO()
			deltaTx := currTx - prevTx
			deltaRx := currRx - prevRx

			txKbps := (float64(deltaTx) * 8 / 1000) / float64(sampleInterval)
			rxKbps := (float64(deltaRx) * 8 / 1000) / float64(sampleInterval)

			memS, err := mem.VirtualMemory()
			var memP float64
			if err != nil {
				log.Println("Memory error:", err)
			} else {
				memP = memS.UsedPercent
			}

			connCount := float64(getActiveConns())

			mu.Lock()
			*cpuH = append(*cpuH, cpuP)
			if len(*cpuH) > maxSamples {
				*cpuH = (*cpuH)[1:]
			}
			*memH = append(*memH, memP)
			if len(*memH) > maxSamples {
				*memH = (*memH)[1:]
			}
			*txH = append(*txH, txKbps)
			if len(*txH) > maxSamples {
				*txH = (*txH)[1:]
			}
			*rxH = append(*rxH, rxKbps)
			if len(*rxH) > maxSamples {
				*rxH = (*rxH)[1:]
			}
			*connH = append(*connH, connCount)
			if len(*connH) > maxSamples {
				*connH = (*connH)[1:]
			}
			mu.Unlock()
		}
	}
}

func average(hist []float64) float64 {
	if len(hist) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range hist {
		sum += v
	}
	return sum / float64(len(hist))
}

func round(val float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(val*shift) / shift
}

func getNetIO() (uint64, uint64) {
	cnts, err := netinfo.IOCounters(true)
	if err != nil {
		log.Println("Net IO error:", err)
		return 0, 0
	}
	var tx, rx uint64
	for _, c := range cnts {
		if strings.HasPrefix(c.Name, "lo") || strings.HasPrefix(c.Name, "docker") || strings.HasPrefix(c.Name, "veth") || strings.HasPrefix(c.Name, "br-") {
			continue
		}
		tx += c.BytesSent
		rx += c.BytesRecv
	}
	return tx, rx
}

func getActiveConns() int {
	conns, err := netinfo.Connections("tcp")
	if err != nil {
		log.Println("Connections error:", err)
		return 0
	}
	count := 0
	for _, c := range conns {
		if c.Status == "ESTABLISHED" {
			count++
		}
	}
	return count
}
