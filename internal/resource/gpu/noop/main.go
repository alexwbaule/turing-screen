package noop

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/metric"
)

// newNoopProvider creates a noop GPU provider that returns zero metrics.
// Logs a warning at startup indicating no GPU metrics will be reported (Req 14.3).
func NewNoopProvider(log *logger.Logger) *Provider {
	log.Warn("GPU provider set to noop: all GPU metrics will report zero values")
	return &Provider{log: log}
}

// noopProvider returns zero metrics for all GPU fields.
type Provider struct {
	log *logger.Logger
}

func (p *Provider) GetMetrics() (*metric.GPU, error) {
	return &metric.GPU{}, nil
}

func (p *Provider) Available() bool {
	return false
}
