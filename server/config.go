package server

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerIP           string   `yaml:"server_ip"`
	StartingIP         string   `yaml:"starting_ip"`
	EndingIP           string   `yaml:"ending_ip"`
	SubnetMask         string   `yaml:"subnet_mask"`
	RouterIP           string   `yaml:"router_ip"`
	LeaseDurationHours int      `yaml:"lease_duration_hours"`
	DNSServers         []string `yaml:"dns_servers"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("[ERROR] Config file (%s) not found", filename)
		return nil, err
	}

	cfg := Config{}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("[ERROR] File (%s) could not be unmarshalled. Check the file syntax", filename)
		return nil, err
	}

	return &cfg, nil
}
