package server

import (
	"fmt"
	"time"

	"github.com/kyverno/policy-server/pkg/storage"
	"k8s.io/component-base/metrics"
)

// RegisterMetrics registers
func RegisterMetrics(r metrics.KubeRegistry, metricResolution time.Duration) error {
	// register metrics server components metrics
	err := RegisterServerMetrics(r.Register, metricResolution)
	if err != nil {
		return fmt.Errorf("unable to register server metrics: %v", err)
	}
	err = storage.RegisterStorageMetrics(r.Register)
	if err != nil {
		return fmt.Errorf("unable to register storage metrics: %v", err)
	}

	return nil
}
