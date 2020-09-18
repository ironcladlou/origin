package test

import (
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/metrics"
)

var (
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "graceful_test_request_counter",
			Help: "Monotonic count of request",
		},
		[]string{"code", "method",},
	)

	latency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "graceful_test_latency",
			Help: "request result latency",
		},
		[]string{"verb",},
	)
)

func Count(code, method string) prometheus.Counter {
	return counter.WithLabelValues(code, method)
}

func Latency(verb string) prometheus.Gauge {
	return latency.WithLabelValues(verb)
}

func ClientGoMetricsInitialize() error {
	if err := prometheus.Register(counter); err != nil {
		return err
	}

	if err := prometheus.Register(latency); err != nil {
		return err
	}

	metrics.Register(metrics.RegisterOpts{
		RequestLatency: LatencyMetricFunc(Observe),
		RequestResult: ResultMetricFunc(Increment),
	})
	return nil
}

// ResultMetric counts response codes partitioned by method and host.
type ResultMetricFunc func(code string, method string, host string)

func (f ResultMetricFunc) Increment(code string, method string, host string) {
	f(code, method, host)
}

// LatencyMetric observes client latency partitioned by verb and url.
type LatencyMetricFunc func(verb string, u url.URL, latency time.Duration)

func (f LatencyMetricFunc) Observe(verb string, u url.URL, latency time.Duration) {
	f(verb, u, latency)
}

func Increment(code string, method string, host string) {
	Count(code, method).Inc()
}

func Observe(verb string, u url.URL, latency time.Duration) {
	Latency(verb).Set(float64(latency.Milliseconds()))
}






