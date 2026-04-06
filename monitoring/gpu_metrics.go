// Package monitoring collects per-container GPU metrics and emits them to
// the Loggregator v2 firehose.
//
// Real implementation note:
// GPU metrics (utilisation, memory) should be read via the NVML library
// (github.com/NVIDIA/go-nvml) on the cell VM.  This prototype simulates
// metric values so the package compiles and the data flow can be tested
// without NVML present.  Replace the Collect implementation with NVML calls
// once the package is deployed on a GPU cell.
package monitoring

import (
	"math/rand"
	"time"
)

// GPUManager is a local interface satisfied by executor.GPUManager, keeping
// this package self-contained without a hard import cycle.
type GPUManager interface {
	TotalGPUs() int
}

// GPUMetric carries a single GPU performance sample for one container.
type GPUMetric struct {
	// ContainerHandle is the garden container handle (== CF app instance GUID).
	ContainerHandle string

	// GPUIndex is the 0-based device index on the host.
	GPUIndex uint

	// UtilizationPercent is the GPU compute utilisation in percent (0–100).
	UtilizationPercent float64

	// MemoryUsedMiB is the amount of device memory currently in use (MiB).
	MemoryUsedMiB uint64

	// MemoryTotalMiB is the total device memory capacity (MiB).
	MemoryTotalMiB uint64

	// Timestamp is the wall-clock time at which the sample was taken.
	Timestamp time.Time
}

// GPUMetricsCollector gathers GPU metrics from allocated containers.
type GPUMetricsCollector struct{}

// Collect returns a simulated slice of GPUMetric values for the GPUs tracked
// by manager.  In a production implementation this method would:
//
//  1. Call nvml.DeviceGetHandleByIndex(i) for each GPU.
//  2. Call nvml.DeviceGetUtilizationRates(handle) for utilisation.
//  3. Call nvml.DeviceGetMemoryInfo(handle) for memory figures.
//  4. Associate each reading with the container handle via the allocation
//     table in the GPUManager.
//
// The appGUID parameter is used to tag the metrics so that Loggregator can
// route them to the correct app's log stream.
func (c *GPUMetricsCollector) Collect(manager GPUManager) []GPUMetric {
	total := manager.TotalGPUs()
	metrics := make([]GPUMetric, 0, total)
	now := time.Now()

	for i := 0; i < total; i++ {
		// Simulated values – replace with NVML calls in production.
		metrics = append(metrics, GPUMetric{
			ContainerHandle:    "",
			GPUIndex:           uint(i),
			UtilizationPercent: rand.Float64() * 100, //nolint:gosec // simulation only
			MemoryUsedMiB:      uint64(rand.Intn(16000)), //nolint:gosec // simulation only
			MemoryTotalMiB:     16160,
			Timestamp:          now,
		})
	}
	return metrics
}

// EmitToLoggregator forwards GPU metrics to the Loggregator agent as v2
// gauge envelopes.
//
// Integration notes:
//
//   - Use the Loggregator v2 client (code.cloudfoundry.org/go-loggregator/v9)
//     to create an IngressClient connected to the local Loggregator agent gRPC
//     endpoint (unix:///var/vcap/run/loggregator-agent/loggregator_v2.sock).
//
//   - Each GPUMetric becomes one v2 Envelope with type Gauge containing:
//       "gpu_utilization"  value=metric.UtilizationPercent  unit="percent"
//       "gpu_memory_used"  value=float64(metric.MemoryUsedMiB)  unit="MiB"
//       "gpu_memory_total" value=float64(metric.MemoryTotalMiB) unit="MiB"
//
//   - Tags on the envelope:
//       source_id   = appGUID
//       instance_id = containerHandle
//       gpu_index   = strconv.Itoa(int(metric.GPUIndex))
//
//   - The Loggregator agent batches and forwards the envelopes to Log Cache
//     where they are queryable via `cf log-cache myapp --envelope-type gauge`.
//
// This stub logs to stdout so the prototype can be exercised without a live
// Loggregator deployment.
func (c *GPUMetricsCollector) EmitToLoggregator(metrics []GPUMetric, appGUID string) {
	for _, m := range metrics {
		_ = appGUID
		_ = m
		// TODO: replace with loggregator v2 IngressClient.Send() call.
	}
}
