package utils

import (
	"bufio" // For efficient file writing
	"fmt"
	"log"
	"net"
	"os"
	"strconv" // Used by the Print() function, so we keep it
	"time"
	// "encoding/csv" // No longer needed for the new ExportCsv function
)

const (
	defaultOutput         = "result.csv" // User can choose another name like clean_ips.txt with -o
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

// NoPrintResult checks if test results should be printed to the console.
func NoPrintResult() bool {
	return PrintNum == 0
}

// noOutput checks if file output should be skipped.
func noOutput() bool {
	return Output == "" || Output == " "
}

// PingData holds basic ping test results for an IP.
type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

// CloudflareIPData extends PingData with Cloudflare-specific metrics like download speed.
type CloudflareIPData struct {
	*PingData
	lossRate      float32
	DownloadSpeed float64 // Bytes per second
}

// getLossRate calculates and returns the packet loss rate.
// It caches the result after the first calculation.
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 { // Only calculate if not already calculated
		if cf.Sended > 0 { // Prevent division by zero
			pingLost := cf.Sended - cf.Received
			cf.lossRate = float32(pingLost) / float32(cf.Sended)
		} else {
			cf.lossRate = 1.0 // Consider 100% loss if no pings were sent
		}
	}
	return cf.lossRate
}

// toString converts CloudflareIPData to a slice of strings for console printing.
// This function remains unchanged as it's used by Print().
func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 6)
	if cf.IP != nil {
		result[0] = cf.IP.String()
	} else {
		result[0] = "N/A" // In case IP is nil for some reason
	}
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = strconv.FormatFloat(float64(cf.getLossRate()), 'f', 2, 32) // Loss rate
	result[4] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)   // Delay in ms
	result[5] = strconv.FormatFloat(cf.DownloadSpeed/1024/1024, 'f', 2, 32) // Download speed in MB/s
	return result
}

// ExportCsv is the modified function to save only clean IPs, each on a new line.
func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	// 'Output' variable gets the filename from the -o flag.
	// User can specify a filename like "clean_ips.txt".
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Failed to create file [%s]: %v", Output, err)
		return
	}
	defer fp.Close()

	writer := bufio.NewWriter(fp) // Use bufio.Writer for more efficient writing

	for _, v := range data {
		if v.IP != nil { // Ensure IP is not nil.
			_, err := writer.WriteString(v.IP.String() + "\n") // Write IP and a newline character
			if err != nil {
				log.Printf("Error writing IP to file: %v", err)
			}
		}
	}

	err = writer.Flush() // Very important: flush the buffer to write all data to the file
	if err != nil {
		log.Printf("Error flushing writer: %v", err)
	}

	log.Printf("Successfully wrote clean IPs (one per line) to %s", Output)
}

// convertToString is used by Print() and remains unchanged.
func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// PingDelaySet is a slice of CloudflareIPData for sorting by delay and loss rate.
type PingDelaySet []CloudflareIPData

// FilterDelay filters the PingDelaySet based on InputMinDelay and InputMaxDelay.
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay { // When the input delay condition is not within the default range, no filtering is performed
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay { // When the input delay condition is the default value, no filtering is performed
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay { // Upper limit of average delay, when the delay is greater than the maximum value of the condition, no subsequent data meets the condition, directly exit the loop
			break
		}
		if v.Delay < InputMinDelay { // Lower limit of average delay, when the delay is less than the minimum value of the condition, it does not meet the condition, skip
			continue
		}
		data = append(data, v) // When the delay meets the condition, add it to the new array
	}
	return
}

// FilterLossRate filters the PingDelaySet based on InputMaxLossRate.
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate { // When the input packet loss condition is the default value, no filtering is performed
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // Upper limit of packet loss rate
			break
		}
		data = append(data, v) // When the packet loss rate meets the condition, add it to the new array
	}
	return
}

// Len is part of sort.Interface.
func (s PingDelaySet) Len() int {
	return len(s)
}

// Less is part of sort.Interface. It sorts by loss rate first, then by delay.
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate // Lower loss rate is better
	}
	return s[i].Delay < s[j].Delay // Lower delay is better
}

// Swap is part of sort.Interface.
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// DownloadSpeedSet is a slice of CloudflareIPData for sorting by download speed.
type DownloadSpeedSet []CloudflareIPData

// Len is part of sort.Interface.
func (s DownloadSpeedSet) Len() int {
	return len(s)
}

// Less is part of sort.Interface. It sorts by download speed (descending).
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed // Higher download speed is better
}

// Swap is part of sort.Interface.
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Print displays the results in the console. This function remains unchanged
// and will still display all details for the specified number of IPs.
func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 { // When the length of the IP array (number of IPs) is 0, skip outputting results
		fmt.Println("\n[Info] The number of complete test results IP is 0, skipping output results.")
		return
	}
	dateString := convertToString(s) // Convert to multi-dimensional array [][]string
	if len(dateString) < PrintNum {  // If the length of the IP array (number of IPs) is less than the printing times, change the times to the number of IPs
		PrintNum = len(dateString)
	}
	headFormat := "%-15s%-5s%-9s%-10s%-14s%-21s\n"
	dataFormat := "%-17s%-7s%-7s%-13s%-15s%-15s\n"
	for i := 0; i < PrintNum; i++ { // If the IPs to be output contain IPv6, adjust the spacing
		// Add a check to prevent panic if dateString[i] doesn't have enough elements (should not happen with current toString)
		if len(dateString[i]) > 0 && len(dateString[i][0]) > 15 {
			headFormat = "%-40s%-5s%-9s%-10s%-14s%-21s\n"
			dataFormat = "%-42s%-7s%-7s%-13s%-15s%-15s\n"
			break
		}
	}
	fmt.Printf(headFormat, "IP Address", "Sent", "Received", "Loss-Rate", "Average-Delay", "Download-Speed (MB/s)")
	for i := 0; i < PrintNum; i++ {
		// Add a check to prevent panic if dateString[i] doesn't have enough elements
		if len(dateString[i]) == 6 {
			fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5])
		} else {
			log.Printf("Warning: Malformed data for printing at index %d", i)
		}
	}
	if !noOutput() {
		// Adjust console message to reflect that the output file format has changed.
		fmt.Printf("\nClean IPs (one per line) have been written to %v file.\n", Output)
		if Output == defaultOutput { // If the output filename is the default result.csv
			fmt.Printf("Note: The file %s now contains only IP addresses, one per line, not full CSV data.\n", Output)
		}
		fmt.Println("Full details are shown above in the console (if PrintNum > 0).")
	}
}
