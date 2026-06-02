package interfaces

import "github.com/alexwbaule/turing-screen/internal/domain/entity/metric"

type Provider interface {
	// GetMetrics returns the current GPU metrics.
	GetMetrics() (*metric.GPU, error)
	// Available reports whether the GPU provider has a usable GPU.
	Available() bool
}
