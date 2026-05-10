package server

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// SaveLeases saves the leases in the current state to the json file "lease_file" as configured in the config.yaml
func (r *IPPool) SaveLeases() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := filepath.Dir(r.LeaseJSONfile)
	os.MkdirAll(dir, 0o755)

	data, err := json.MarshalIndent(r.LeaseMap, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to save leases: %v", err)
		return err
	}
	return os.WriteFile(r.LeaseJSONfile, data, 0o644)
}

// LoadLeases loads the previously allocated leases from the json file "lease_file" upon server start
func (r *IPPool) LoadLeases() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.LeaseJSONfile)
	if err != nil {
		log.Printf("[ERROR] Failed to load leases: %v", err)
		return err
	}
	return json.Unmarshal(data, &r.LeaseMap)
}
