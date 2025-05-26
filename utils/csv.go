package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	defaultOutput = "result.csv"
	maxDelay      = 9999 * time.Millisecond
	minDelay      = 0 * time.Millisecond
)

var (
	Output   = defaultOutput
	PrintNum = 10
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
	IP    *net.IPAddr
	Delay time.Duration
}

type CloudflareIPData struct {
	IP *net.IPAddr
	// Add other fields here if needed
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 1)
	result[0] = cf.IP.String()
	return result
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Failed to create file [%s]: %v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp)
	_ = w.WriteAll(convertToString(data))
	w.Flush()
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

func printResults(s []CloudflareIPData) {
	if NoPrintResult() {
		return
	}
	if len(s) == 0 {
		fmt.Println("\n[Info] The number of complete test result IPs is 0, skipping output.")
		return
	}
	dateString := convertToString(s)
	if len(dateString) < PrintNum {
		PrintNum = len(dateString)
	}
	fmt.Printf("%-20s\n", "IP Address")
	for i := 0; i < PrintNum; i++ {
		fmt.Printf("%-20s\n", dateString[i][0])
	}
	if !noOutput() {
		fmt.Printf("\nComplete test results have been written to %v file.\n", Output)
	}
}
