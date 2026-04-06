// Package api provides the CAPI HTTP handlers for GPU process resources.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/johha/cf-gpuccino/capi/models"
)

// GPUHandler handles HTTP requests for GPU resource management on CF processes.
type GPUHandler struct {
	// currentResources holds the in-memory state for the prototype.
	// A real CAPI implementation would read/write from the CC database.
	currentResources models.GPUResources
}

// UpdateProcessGPU handles PATCH /v3/processes/:guid/gpu
// It reads a JSON body containing gpu and gpu_type fields, validates them,
// updates the in-memory state, and responds with the updated resource JSON.
//
// Responses:
//   - 200 OK         – update accepted; body contains the updated GPU resources.
//   - 400 Bad Request – body could not be decoded as JSON.
//   - 422 Unprocessable Entity – validation error (invalid gpu_type, negative count, …).
func (h *GPUHandler) UpdateProcessGPU(w http.ResponseWriter, r *http.Request) {
	var msg models.ProcessGPUUpdateMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, `{"errors":[{"detail":"Request body is not valid JSON"}]}`, http.StatusBadRequest)
		return
	}

	if err := msg.Validate(); err != nil {
		http.Error(w,
			`{"errors":[{"detail":"`+err.Error()+`"}]}`,
			http.StatusUnprocessableEntity,
		)
		return
	}

	h.currentResources = msg.GPUResources

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(h.currentResources); err != nil {
		// The response header is already written; log the error for operators.
		_ = err // TODO: replace with structured logger (e.g. lager) in production.
	}
}

// GetProcessGPU handles GET /v3/processes/:guid/gpu
// It returns the current GPU resource configuration as JSON.
//
// Responses:
//   - 200 OK – body contains the current GPU resources.
func (h *GPUHandler) GetProcessGPU(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(h.currentResources); err != nil {
		// The response header is already written; log the error for operators.
		_ = err // TODO: replace with structured logger (e.g. lager) in production.
	}
}
