// Package garden extends the garden-runc ContainerSpec with GPU device support.
//
// There are two approaches for exposing GPU devices to a container:
//
//  1. Legacy device approach: explicitly list the host device nodes
//     (/dev/nvidia0, /dev/nvidiactl, /dev/nvidia-uvm) together with
//     their major/minor numbers in the OCI Linux.Devices and
//     Linux.Resources.Devices sections. This works but is fragile –
//     device numbers can vary between hosts.
//
//  2. CDI (Container Device Interface) approach: reference a named CDI
//     device (e.g. "nvidia.com/gpu=0"). The CDI registry (driven by
//     JSON spec files under /etc/cdi/ or /var/run/cdi/) resolves the
//     name into the full set of device nodes, mounts, and environment
//     variables required by that device. This is the preferred approach
//     for new deployments.
//
// This file defines the types used to carry both representations through
// the garden API, and provides a helper (GPUContainerConfig) that builds
// the lists from a GPURequest.
package garden

import (
	"fmt"
	"strconv"
	"strings"
)

// DeviceSpec describes a single host device to be exposed inside the container.
// It maps to the OCI Linux.Devices entry.
type DeviceSpec struct {
	// Type is the device type: "c" (char), "b" (block), or "p" (pipe).
	Type string

	// Path is the absolute device path inside the container (e.g. /dev/nvidia0).
	Path string

	// Major is the device major number on the host.
	Major int64

	// Minor is the device minor number on the host.
	Minor int64

	// Permissions is the cgroup device rule permission string, e.g. "rwm".
	Permissions string
}

// CDIDevice references a device by its CDI fully-qualified name.
// The garden-runc backend passes this to the OCI runtime which resolves
// it via the CDI registry before starting the container.
// Format: "<vendor>/<class>=<name>", e.g. "nvidia.com/gpu=0".
type CDIDevice struct {
	// Name is the CDI fully-qualified device name.
	Name string
}

// ContainerSpec is a garden ContainerSpec extended with GPU device fields.
// In a real integration these fields would be added to the upstream
// garden ContainerSpec struct; here we define our own type for the prototype.
type ContainerSpec struct {
	// Handle is the unique container identifier.
	Handle string

	// RootFSPath is the path to the root filesystem image.
	RootFSPath string

	// CDIDevices lists CDI device names to inject via the CDI registry.
	// Preferred over Devices when the CDI runtime hook is available.
	CDIDevices []CDIDevice

	// Devices lists explicit host device nodes to bind-mount into the container.
	// Used as a fallback when CDI is not available.
	Devices []DeviceSpec

	// Env holds additional environment variables to set in the container.
	Env []string
}

// GPUContainerConfig builds the CDIDevices and Devices slices for a
// ContainerSpec from a GPURequest.  gpuIndices must already be resolved
// (i.e. the executor has called GPUManager.Allocate before this helper).
//
// When useCDI is true the function emits CDI device names only.
// When useCDI is false it falls back to the legacy DeviceSpec approach
// using well-known device paths and typical major/minor numbers for NVIDIA.
func GPUContainerConfig(gpuType string, gpuIndices []uint, useCDI bool) ContainerSpec {
	spec := ContainerSpec{}

	for _, idx := range gpuIndices {
		if useCDI {
			// CDI approach: single structured name; the runtime resolves
			// all necessary device nodes and mounts automatically.
			spec.CDIDevices = append(spec.CDIDevices, CDIDevice{
				Name: fmt.Sprintf("%s.com/gpu=%d", gpuType, idx),
			})
		} else {
			// Legacy approach: manually enumerate required device nodes.
			// NVIDIA major number is 195; minor 255 = nvidiactl, minor N = /dev/nvidiaN.
			// nvidia-uvm uses a dynamic major (here we use a placeholder of 510).
			spec.Devices = append(spec.Devices,
				DeviceSpec{
					Type:        "c",
					Path:        fmt.Sprintf("/dev/nvidia%d", idx),
					Major:       195,
					Minor:       int64(idx),
					Permissions: "rwm",
				},
				DeviceSpec{
					Type:        "c",
					Path:        "/dev/nvidiactl",
					Major:       195,
					Minor:       255,
					Permissions: "rwm",
				},
				DeviceSpec{
					Type:        "c",
					Path:        "/dev/nvidia-uvm",
					Major:       510,
					Minor:       0,
					Permissions: "rwm",
				},
			)
		}
	}

	// CUDA_VISIBLE_DEVICES expects a comma-separated list of device indices
	// (e.g. "0,1") in a single environment variable – not one entry per GPU.
	if len(gpuIndices) > 0 {
		idxStrs := make([]string, len(gpuIndices))
		for i, idx := range gpuIndices {
			idxStrs[i] = strconv.FormatUint(uint64(idx), 10)
		}
		spec.Env = append(spec.Env,
			"CUDA_VISIBLE_DEVICES="+strings.Join(idxStrs, ","),
		)
	}

	return spec
}
