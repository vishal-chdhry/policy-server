package server

import (
	"time"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // for client-go metrics registration
)

type Config struct {
	Apiserver        *genericapiserver.Config
	Rest             *rest.Config
	MetricResolution time.Duration
}
