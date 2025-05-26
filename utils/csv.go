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
)

var (
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
 Delay    time.Duration
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
 w := csv.NewWriter(fp) // Create a new file writing stream
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

func () print() [
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
  if len(dateString[i][0]) > 15 {
   headFormat = "%-40s%-5s%-9s%-10s%-14s%-21s\n"
   dataFormat = "%-42s%-7s%-7s%-13s%-15s%-15s\n"
   break
  }
 }
 fmt.Printf(headFormat, "IP Address")
 for i := 0; i < PrintNum; i++ {
  fmt.Printf(dataFormat, dateString[i][0])
 }
 if !noOutput() {
  fmt.Printf("\nComplete test results have been written to %v file, which can be viewed using Notepad/Spreadsheet software.\n", Output)
 }
    ]
