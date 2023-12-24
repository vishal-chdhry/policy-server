package server

import (
	"context"
	"net/http"
	"time"

	"github.com/kyverno/policy-server/pkg/utils"
	"k8s.io/component-base/metrics"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
)

var (
	// initialized below to an actual value by a call to RegisterTickDuration
	// (acts as a no-op by default), but we can't just register it in the constructor,
	// since it could be called multiple times during setup.
	tickDuration = metrics.NewHistogram(&metrics.HistogramOpts{})
)

// RegisterServerMetrics creates and registers a histogram metric for
// scrape duration.
func RegisterServerMetrics(registrationFunc func(metrics.Registerable) error, resolution time.Duration) error {
	tickDuration = metrics.NewHistogram(
		&metrics.HistogramOpts{
			Namespace: "policy_server",
			Subsystem: "manager",
			Name:      "tick_duration_seconds",
			Help:      "The total time spent collecting and storing polices in seconds.",
			Buckets:   utils.BucketsForScrapeDuration(resolution),
		},
	)
	return registrationFunc(tickDuration)
}

func NewServer(
	apiserver *genericapiserver.GenericAPIServer,
	// storage storage.Storage,
) *server {
	return &server{
		GenericAPIServer: apiserver,
		// storage:          storage,
	}
}

type server struct {
	*genericapiserver.GenericAPIServer
	// storage    storage.Storage
}

// RunUntil starts background scraping goroutine and runs apiserver serving metrics.
func (s *server) RunUntil(stopCh <-chan struct{}) error {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// store
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}

func (s *server) RegisterProbes() error {
	err := s.AddReadyzChecks(s.probeMetricStorageReady("metric-storage-ready"))
	if err != nil {
		return err
	}
	return nil
}

func (s *server) probeMetricStorageReady(name string) healthz.HealthChecker {
	return healthz.NamedCheck(name, func(r *http.Request) error {
		// if !s.storage.Ready() {
		// 	err := fmt.Errorf("no metrics to serve")
		// 	klog.InfoS("Failed probe", "probe", name, "err", err)
		// 	return err
		// }
		return nil
	})
}
