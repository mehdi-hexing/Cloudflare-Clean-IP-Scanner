package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
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
	lossRate      float32
	DownloadSpeed float64
}

// Calculate packet loss rate
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.Sended == 0 {
		return 0.0
	}
	// Re-calculate lossRate if it's zero (or not pre-set) and Sended is positive.
	// This assumes lossRate might be explicitly set to 0 to force recalculation,
	// or it's the first time being accessed.
	// If lossRate could be non-zero from an external source and shouldn't be overridden,
	// this logic might need adjustment. For now, it prioritizes calculation if lossRate is zero.
	if cf.lossRate == 0.0 && cf.Received == 0 && cf.Sended > 0 { // If explicitly set to 0 for recalc, or initial calc
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	} else if cf.Sended > 0 && (cf.Sended-cf.Received)/cf.Sended != int(cf.lossRate) {
		// If loss rate was previously calculated but Sended/Received changed, recalculate
		// This part is tricky: cf.lossRate is float32.
		// A simpler way: always calculate unless it's been specifically set to a non-zero value
		// that we want to preserve. Assuming it's always derived from Sended/Received here.
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

// toString prepares a string slice for console output.
func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 6)
	if cf.IP != nil {
		result[0] = cf.IP.String()
	} else {
		result[0] = "N/A"
	}
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = fmt.Sprintf("%.2f%%", cf.getLossRate()*100)
	result[4] = fmt.Sprintf("%dms", cf.Delay.Milliseconds())
	result[5] = fmt.Sprintf("%.2f", cf.DownloadSpeed)
	return result
}

// convertToString converts a slice of CloudflareIPData to a 2D string slice for console display.
func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0, len(data))
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// convertToIPOnlyCsvData converts a slice of CloudflareIPData to a 2D string slice
// where each inner slice contains only the IP address, for CSV export.
func convertToIPOnlyCsvData(data []CloudflareIPData) [][]string {
	result := make([][]string, 0, len(data))
	for _, v := range data {
		if v.IP != nil {
			result = append(result, []string{v.IP.String()})
		} else {
			result = append(result, []string{"N/A"})
		}
	}
	return result
}

// ExportCsv writes ONLY the IP addresses to the specified output file, one IP per line.
func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output) // os.Create truncates if file exists
	if err != nil {
		log.Fatalf("Failed to create file [%s]: %v", Output, err)
	}
	defer fp.Close()

	w := csv.NewWriter(fp)

	// NO HEADER IS WRITTEN as per user request.

	csvData := convertToIPOnlyCsvData(data)
	if err := w.WriteAll(csvData); err != nil {
		log.Printf("Failed to write all data to CSV file [%s]: %v", Output, err)
		// It's good to let the user know if the write failed, but `log.Fatalf` might be too harsh if some data was written.
		// However, `WriteAll` attempts to write all records, so if it fails, it's usually a more significant issue.
	}
	w.Flush()
	if err := w.Error(); err != nil { // Check for errors after flush
		log.Printf("Error flushing CSV writer for [%s]: %v", Output, err)
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
	data = make(PingDelaySet, 0)
	for _, v := range s {
		if v.Delay > InputMaxDelay {
			continue // If not sorted by delay, must check all.
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
	data = make(PingDelaySet, 0)
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // Assumes PingDelaySet is sorted by loss rate first
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

func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 {
		fmt.Println("\n[Info] The number of complete test results IP is 0, skipping output results.")
		return
	}

	dateString := convertToString(s)
	displayCount := PrintNum
	if len(dateString) < PrintNum {
		displayCount = len(dateString)
	}
	
	if displayCount == 0 && len(s) > 0 {
		// This case implies PrintNum was 0, which should have been caught by NoPrintResult().
		// Or, PrintNum is > 0 but len(dateString) is 0, caught by len(s) <= 0.
		// So, if displayCount is 0 here, it means no results to print or printing is disabled.
		// The len(s) check above handles "no results". NoPrintResult() handles "printing disabled".
		return
	}


	headFormat := "%-15s%-8s%-9s%-10s%-14s%-21s\n"
	dataFormat := "%-17s%-8s%-9s%-13s%-15s%-15s\n"

	isIPv6Present := false
	if displayCount > 0 {
		for i := 0; i < displayCount; i++ {
			// Ensure dateString[i] exists and has at least one element before checking its length
			if len(dateString[i]) > 0 && len(dateString[i][0]) > 15 {
				isIPv6Present = true
				break
			}
		}
	}

	if isIPv6Present {
		headFormat = "%-40s%-8s%-9s%-10s%-14s%-21s\n"
		dataFormat = "%-42s%-8s%-9s%-13s%-15s%-15s\n"
	}

	fmt.Printf(headFormat, "IP Address", "Sent", "Received", "Loss-Rate", "Avg-Delay", "Download-Speed (MB/s)")
	for i := 0; i < displayCount; i++ {
		if len(dateString[i]) == 6 {
			fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5])
		} else {
			// This should ideally not happen if data is consistent
			if len(dateString[i]) > 0 {
				log.Printf("[Error] Malformed data string for IP (expected 6 fields): %s", dateString[i][0])
			} else {
				log.Printf("[Error] Malformed data string: empty record at index %d", i)
			}
		}
	}

	if !noOutput() {
		// Updated message to accurately reflect that only a list of IPs is saved.
		fmt.Printf("\n[Info] A list of tested IP addresses has been written to the %v file (one IP per line).\n", Output)
	}
}
