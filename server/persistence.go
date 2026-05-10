package server

import (
	"encoding/json"
	"os"
)

// SaveLeases saves the leases in the current state to the json file "lease_file" as configured in the config.yaml
func (r *IPPool) SaveLeases() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(r.LeaseMap, "", "  ")
	if err != nil {
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
		return err
	}
	return json.Unmarshal(data, &r.LeaseMap)
}
