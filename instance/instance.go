package instance

import (
	"encoding/json"

	"github.com/namely/broadway/env"
)

// Instance entity
type Instance struct {
	PlaybookID string            `json:"playbook_id" binding:"required"`
	ID         string            `json:"id"`
	Created    string            `json:"created"`
	Vars       map[string]string `json:"vars"`
	Status     `json:"status"`
}

// Status for an instance
type Status string

const (
	// StatusNew represents a newly created instance
	StatusNew Status = ""
	// StatusDeploying represents an instance that has begun deployment
	StatusDeploying = "deploying"
	// StatusDeployed represents an instance that has been deployed successfully
	StatusDeployed = "deployed"
	// StatusDeleting represents an instance that has begun deltion
	StatusDeleting = "deleting"
	// StatusError represents an instance that broke
	StatusError = "error"
)

// JSON instance representation
func (i *Instance) JSON() (string, error) {
	encoded, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// Path for an instance
func (i *Instance) Path() string {
	return env.EtcdPath + "/instances/" + i.PlaybookID + "/" + i.ID
}
