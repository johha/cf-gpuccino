// Package models contains the CAPI (Cloud Controller API) data models for
// GPU resource requests on CF v3 processes.
package models

import (
	"errors"
	"strings"
)

// GPUResources holds the GPU-related fields that can be set on a CF process.
// These map to the resources section of the CF v3 process API.
type GPUResources struct {
	// GPULimit is the number of GPU devices requested (0 = no GPU).
	GPULimit int `json:"gpu"`

	// GPUType specifies the preferred GPU vendor: "nvidia", "amd", or "" (any).
	GPUType string `json:"gpu_type,omitempty"`
}

// ProcessGPUUpdateMessage is the request body accepted by
// PATCH /v3/processes/:guid for GPU resource changes.
// It embeds GPUResources so the JSON fields are promoted to the top level,
// matching the flat CF v3 resource update convention.
type ProcessGPUUpdateMessage struct {
	GPUResources
}

// Validate checks that the GPU resource request is semantically valid.
// Returns a non-nil error if the values are out of range or unrecognised.
func (m *ProcessGPUUpdateMessage) Validate() error {
	if m.GPULimit < 0 {
		return errors.New("gpu must be a non-negative integer")
	}

	switch strings.ToLower(m.GPUType) {
	case "", "nvidia", "amd":
		// valid values
	default:
		return errors.New("gpu_type must be one of: \"nvidia\", \"amd\", or omitted")
	}

	return nil
}
