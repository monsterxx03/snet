// +build ignore

package main

import (
	"bufio"
	"os"
	"strings"
)

var adHostFile = "ad_hosts.txt"
var resultFIle = "ad_hosts"
var whiteListDomains = map[string]int{"analytics.google.com": 1, "0.0.0.0": 1}

func genAdHosts() ([]string, error) {
	f, err := os.Open(adHostFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	result := make([]string, 0, 37000)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] != "0.0.0.0" {
			continue
		}
		domain := fields[1]
		if _, ok := whiteListDomains[domain]; ok {
			continue
		}
		result = append(result, domain)
	}
	return result, err
}

func main() {
	result, err := genAdHosts()
	if err != nil {
		panic(err)
	}
	f, err := os.Create("ad_hosts")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(strings.Join(result, "\n"))
}
