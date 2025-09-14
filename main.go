package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the DNS updater
type Config struct {
	DOToken    string
	Domain     string
	Subdomain  string
	RecordType string
	TTL        int
}

// IPResponse represents the response from api.myip.com
type IPResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	CC      string `json:"cc"`
}

// DORecord represents a Digital Ocean DNS record
type DORecord struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

// DORecordsResponse represents the response from DO API
type DORecordsResponse struct {
	DomainRecords []DORecord `json:"domain_records"`
}

// DOUpdateRequest represents the request to update a DNS record
type DOUpdateRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

const (
	myIPURL  = "https://api.myip.com"
	doAPIURL = "https://api.digitalocean.com/v2"
)

func main() {
	config, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config from environment: %v", err)
	}

	currentIP, err := getCurrentIP()
	if err != nil {
		log.Fatalf("Failed to get current IP: %v", err)
	}

	fmt.Printf("Current IP: %s\n", currentIP)

	recordID, currentDNSIP, err := getDNSRecord(config)
	if err != nil {
		log.Fatalf("Failed to get DNS record: %v", err)
	}

	if currentDNSIP == currentIP {
		fmt.Printf("IP hasn't changed, no update needed (current: %s)\n", currentIP)
		return
	}

	fmt.Printf("IP changed from %s to %s, updating DNS record...\n", currentDNSIP, currentIP)

	err = updateDNSRecord(config, recordID, currentIP)
	if err != nil {
		log.Fatalf("Failed to update DNS record: %v", err)
	}

	fmt.Printf("Successfully updated %s.%s to %s\n", config.Subdomain, config.Domain, currentIP)
}

func loadConfigFromEnv() (*Config, error) {
	doToken := os.Getenv("DO_TOKEN")
	if doToken == "" {
		return nil, fmt.Errorf("DO_TOKEN environment variable is required")
	}

	domain := os.Getenv("DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("DOMAIN environment variable is required")
	}

	subdomain := os.Getenv("SUBDOMAIN")
	if subdomain == "" {
		return nil, fmt.Errorf("SUBDOMAIN environment variable is required")
	}

	recordType := os.Getenv("RECORD_TYPE")
	if recordType == "" {
		recordType = "A" // default to A record
	}

	ttlStr := os.Getenv("TTL")
	ttl := 300 // default TTL
	if ttlStr != "" {
		parsedTTL, err := strconv.Atoi(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TTL value '%s': %v", ttlStr, err)
		}
		ttl = parsedTTL
	}

	config := &Config{
		DOToken:    doToken,
		Domain:     domain,
		Subdomain:  subdomain,
		RecordType: recordType,
		TTL:        ttl,
	}

	fmt.Printf("Configuration loaded:\n")
	fmt.Printf("  Domain: %s\n", config.Domain)
	fmt.Printf("  Subdomain: %s\n", config.Subdomain)
	fmt.Printf("  Record Type: %s\n", config.RecordType)
	fmt.Printf("  TTL: %d\n", config.TTL)
	fmt.Printf("  Token: %s***%s\n", config.DOToken[:8], config.DOToken[len(config.DOToken)-4:])

	return config, nil
}

func getCurrentIP() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	resp, err := client.Get(myIPURL)
	if err != nil {
		return "", fmt.Errorf("failed to get IP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var ipResp IPResponse
	err = json.Unmarshal(body, &ipResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse IP response: %v", err)
	}

	return ipResp.IP, nil
}

func getDNSRecord(config *Config) (int, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	url := fmt.Sprintf("%s/domains/%s/records", doAPIURL, config.Domain)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+config.DOToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get DNS records: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response: %v", err)
	}
	
	var recordsResp DORecordsResponse
	err = json.Unmarshal(body, &recordsResp)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse DNS records response: %v", err)
	}
	
	// Find the record we want to update
	for _, record := range recordsResp.DomainRecords {
		if record.Type == config.RecordType && record.Name == config.Subdomain {
			return record.ID, record.Data, nil
		}
	}
	
	// Debug: print all available records
	fmt.Println("Available DNS records:")
	for _, record := range recordsResp.DomainRecords {
		fmt.Printf("  Type: %s, Name: '%s', Data: %s\n", record.Type, record.Name, record.Data)
	}
	
	return 0, "", fmt.Errorf("DNS record not found for '%s' (type %s). Check available records above", config.Subdomain, config.RecordType)
}

func updateDNSRecord(config *Config, recordID int, newIP string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	
	updateReq := DOUpdateRequest{
		Type: config.RecordType,
		Name: config.Subdomain,
		Data: newIP,
		TTL:  config.TTL,
	}
	
	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %v", err)
	}
	
	url := fmt.Sprintf("%s/domains/%s/records/%d", doAPIURL, config.Domain, recordID)
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+config.DOToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// debugLog prints debug messages only if DEBUG env var is set to "1" or "true"
func debugLog(format string, args ...interface{}) {
	debug := os.Getenv("DEBUG")
	if debug == "1" || debug == "true" {
		fmt.Printf(format+"\n", args...)
	}
}
