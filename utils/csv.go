package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	// "strconv" // <--- حذف شد، چون استفاده نمی‌شود
	"strings"
	"time"
)

const (
	defaultOutput         = "result.csv"
	maxDelay              = 9999 * time.Millisecond
	minDelay              = 0 * time.Millisecond
	maxLossRate   float32 = 1.0
)

var (
	InputMaxDelay    = maxDelay
	InputMinDelay    = minDelay
	InputMaxLossRate = maxLossRate
	Output           = defaultOutput
	PrintNum         = 10
)

// Check if to print test results
func NoPrintResult() bool {
	return PrintNum == 0
}

// Check if to output to file
func noOutput() bool {
	return Output == "" || Output == " "
}

type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

type CloudflareIPData struct {
	*PingData
	// lossRate      float32 // <--- حذف شد، چون استفاده نمی‌شد و getLossRate آن را مقداردهی نمی‌کرد
	DownloadSpeed float64
}

// Calculate packet loss rate directly.
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.Sended == 0 {
		return 1.0 // 100% loss if no packets sent, not a "clean" IP
	}
	pingLost := cf.Sended - cf.Received
	return float32(pingLost) / float32(cf.Sended)
}

// toString now returns only the IP address.
func (cf *CloudflareIPData) toString() []string {
	return []string{cf.IP.String()}
}

// convertToString now filters for clean IPs (0% loss, Sended > 0)
// and uses the simplified toString method.
func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		// Filter for 0 packet loss and ensure packets were actually sent
		if v.Sended > 0 && v.getLossRate() == 0 {
			result = append(result, v.toString())
		}
	}
	return result
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Failed to create file [%s]: %v", Output, err)
		// return // Unreachable due to log.Fatalf
	}
	defer fp.Close()
	w := csv.NewWriter(fp)
	// Header changed to only "IP Address"
	// It's good practice to check these errors, but for now, ignoring them as per previous code.
	_ = w.Write([]string{"IP Address"})
	// convertToString now returns only clean IPs, each as a single-element string slice.
	_ = w.WriteAll(convertToString(data))
	err = w.Flush() // Check error from Flush
	if err != nil {
		log.Printf("Warning: Failed to flush CSV writer for file [%s]: %v", Output, err)
	}
}

// Delay and packet loss sorting
type PingDelaySet []CloudflareIPData

// Delay condition filtering
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay {
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay {
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay {
			break
		}
		if v.Delay < InputMinDelay {
			continue
		}
		data = append(data, v)
	}
	return
}

// Packet loss condition filtering
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate {
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate {
			break
		}
		data = append(data, v)
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Download speed sorting
type DownloadSpeedSet []CloudflareIPData

func (s DownloadSpeedSet) Len() int {
	return len(s)
}
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed
}
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Print function is simplified to output only clean IP addresses to the console.
func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}

	dateString := convertToString(s)

	if len(dateString) == 0 {
		fmt.Println("\n[Info] The number of clean IP test results is 0, skipping console output.")
		return
	}

	numToPrint := PrintNum
	if len(dateString) < PrintNum {
		numToPrint = len(dateString)
	}

	fmt.Printf("\n%-40s\n", "Clean IP Addresses (Console Output):")
	fmt.Println(strings.Repeat("-", 40))

	for i := 0; i < numToPrint; i++ {
		fmt.Printf("%-40s\n", dateString[i][0])
	}

	if !noOutput() {
		fmt.Printf("\nClean IP results have been written to the %v file.\n", Output)
	}
}
