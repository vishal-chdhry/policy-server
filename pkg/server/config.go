package server

import (
	"net/http"
	"time"

	"github.com/kyverno/policy-server/pkg/api"
	"github.com/kyverno/policy-server/pkg/storage"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // for client-go metrics registration
)

type Config struct {
	Apiserver        *genericapiserver.Config
	Rest             *rest.Config
	MetricResolution time.Duration
}

func (c Config) Complete() (*server, error) {
	// Disable default metrics handler and create custom one
	c.Apiserver.EnableMetrics = true
	metricsHandler, err := c.metricsHandler()
	if err != nil {
		return nil, err
	}
	genericServer, err := c.Apiserver.Complete(nil).New("policy-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}
	genericServer.Handler.NonGoRestfulMux.HandleFunc("/metrics", metricsHandler)

	store := storage.NewStorage()
	if err := api.Install(store, genericServer); err != nil {
		return nil, err
	}

	s := NewServer(
		genericServer,
		store,
	)
	err = s.RegisterProbes()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c Config) metricsHandler() (http.HandlerFunc, error) {
	// Create registry for Policy Server metrics
	registry := metrics.NewKubeRegistry()
	err := RegisterMetrics(registry, c.MetricResolution)
	if err != nil {
		return nil, err
	}
	// Register apiserver metrics in legacy registry
	apimetrics.Register()

	// Return handler that serves metrics from both legacy and Metrics Server registry
	return func(w http.ResponseWriter, req *http.Request) {
		legacyregistry.Handler().ServeHTTP(w, req)
		metrics.HandlerFor(registry, metrics.HandlerOpts{}).ServeHTTP(w, req)
	}, nil
}
