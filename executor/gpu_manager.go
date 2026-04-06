// Package executor implements GPU discovery and allocation for the Diego Executor.
//
// The GPUManager is created once per cell at executor startup.  It discovers
// available GPUs (currently via nvidia-smi), maintains an allocation table
// (GPU index → container handle), and exposes Allocate/Release operations
// that are called by the executor when creating or destroying containers.
package executor

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// GPUInfo describes a single GPU device on the host.
type GPUInfo struct {
	// Index is the 0-based device index (matches /dev/nvidia<Index>).
	Index uint

	// UUID is the globally unique device identifier reported by the driver.
	UUID string

	// Name is the human-readable model name, e.g. "Tesla V100-SXM2-16GB".
	Name string

	// MemoryMiB is the total device memory in mebibytes.
	MemoryMiB uint64
}

// GPUManager tracks GPU devices on a cell and the containers that hold them.
// All exported methods are safe for concurrent use.
type GPUManager struct {
	mu        sync.Mutex
	GPUs      []GPUInfo
	allocated map[uint]string // GPU index → container handle
}

// NewGPUManager discovers GPUs on the host and returns a ready GPUManager.
// If nvidia-smi is not available or returns no GPUs, an empty (but valid)
// manager is returned so non-GPU cells can use the same code path safely.
func NewGPUManager() (*GPUManager, error) {
	gpus, err := discoverGPUs()
	if err != nil {
		// Not fatal: the cell simply has no GPUs.
		gpus = []GPUInfo{}
	}
	return &GPUManager{
		GPUs:      gpus,
		allocated: make(map[uint]string),
	}, nil
}

// discoverGPUs runs nvidia-smi and parses its CSV output.
func discoverGPUs() ([]GPUInfo, error) {
	cmd := exec.Command(
		"nvidia-smi",
		"--query-gpu=index,uuid,name,memory.total",
		"--format=csv,noheader",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi unavailable: %w", err)
	}

	var gpus []GPUInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ", ", 4)
		if len(parts) != 4 {
			continue
		}

		idx, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			continue
		}

		// memory.total is reported as "16160 MiB"; strip the unit.
		memStr := strings.TrimSuffix(strings.TrimSpace(parts[3]), " MiB")
		memMiB, _ := strconv.ParseUint(memStr, 10, 64)

		gpus = append(gpus, GPUInfo{
			Index:     uint(idx),
			UUID:      strings.TrimSpace(parts[1]),
			Name:      strings.TrimSpace(parts[2]),
			MemoryMiB: memMiB,
		})
	}
	return gpus, scanner.Err()
}

// Allocate reserves `count` free GPUs for the given container handle.
// It returns the slice of allocated GPU indices on success.
// Returns an error if fewer than `count` GPUs are available.
func (m *GPUManager) Allocate(containerHandle string, count int) ([]uint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var free []uint
	for _, gpu := range m.GPUs {
		if _, used := m.allocated[gpu.Index]; !used {
			free = append(free, gpu.Index)
			if len(free) == count {
				break
			}
		}
	}

	if len(free) < count {
		return nil, errors.New("insufficient free GPUs")
	}

	for _, idx := range free {
		m.allocated[idx] = containerHandle
	}
	return free, nil
}

// Release frees all GPUs held by the given container handle.
func (m *GPUManager) Release(containerHandle string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for idx, handle := range m.allocated {
		if handle == containerHandle {
			delete(m.allocated, idx)
		}
	}
}

// Available returns the number of GPUs that are currently unallocated.
func (m *GPUManager) Available() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, gpu := range m.GPUs {
		if _, used := m.allocated[gpu.Index]; !used {
			count++
		}
	}
	return count
}

// TotalGPUs returns the total number of GPU devices discovered on this cell.
func (m *GPUManager) TotalGPUs() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.GPUs)
}
