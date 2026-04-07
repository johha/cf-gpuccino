// Package rep extends the Diego Rep with GPU capacity advertisement and
// auction scoring helpers.
//
// Integration with Diego Rep auction:
//
// The Diego Rep's /state endpoint returns a CellState that includes resource
// capacities (CPU, memory, disk).  For GPU support we add a GPUCapacity field
// to that state.  The Auctioneer then calls BidForGPU during the LRP auction
// scoring phase: a cell wins the GPU bid only when it has enough free GPUs of
// the requested type.
//
// In a full integration GPUCapacity would be embedded in diego/rep.CellState
// and BidForGPU would influence the auctioneer's scoring function.  This
// package provides the types and helpers needed for that wiring.
package rep

// GPUManager is a local interface so the rep package remains self-contained
// without a hard import of the executor package.  The executor.GPUManager
// satisfies this interface.
type GPUManager interface {
	Available() int
	TotalGPUs() int
}

// GPUCapacity describes the GPU resources of a Diego cell as reported to
// the Auctioneer during cell state advertisement.
type GPUCapacity struct {
	// TotalGPUs is the total number of GPU devices on this cell.
	TotalGPUs int

	// FreeGPUs is the number of currently unallocated GPU devices.
	FreeGPUs int

	// GPUType identifies the vendor/type of GPUs on this cell,
	// e.g. "nvidia" or "amd".  Cells advertise a single homogeneous type.
	GPUType string
}

// AdvertiseGPUCapacity builds a GPUCapacity snapshot from the live GPUManager
// state.  This is called just before the cell state is serialised and sent to
// the Auctioneer.
func AdvertiseGPUCapacity(manager GPUManager, gpuType string) GPUCapacity {
	return GPUCapacity{
		TotalGPUs: manager.TotalGPUs(),
		FreeGPUs:  manager.Available(),
		GPUType:   gpuType,
	}
}

// BidForGPU returns true when the available capacity satisfies the GPU
// request.  The bid succeeds when:
//  1. The cell has at least `requested` free GPUs.
//  2. The cell's GPU type matches `requestedType`, or `requestedType` is
//     empty (meaning "any type is acceptable").
//
// This function is analogous to the existing memory/disk bid helpers in the
// Diego Rep auction scorer (see rep/auction_cell_rep.go in cloudfoundry/diego-release).
func BidForGPU(available GPUCapacity, requested int, requestedType string) bool {
	if available.FreeGPUs < requested {
		return false
	}
	if requestedType != "" && available.GPUType != requestedType {
		return false
	}
	return true
}
