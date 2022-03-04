package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	global = newMetrics()
)

type Metrics struct {
	services           prometheus.Gauge
	requests           *prometheus.CounterVec
	requestsInFlight   *prometheus.GaugeVec
	requestSeconds     *prometheus.HistogramVec
	requestInputBytes  *prometheus.CounterVec
	requestOutputBytes *prometheus.CounterVec
}

func newMetrics() *Metrics {
	m := &Metrics{
		services: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gost_services",
				Help: "Current number of services",
			}),

		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_requests_total",
				Help: "Total number of requests",
			},
			[]string{"service"}),

		requestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gost_service_requests_in_flight",
				Help: "Current in-flight requests",
			},
			[]string{"service"}),

		requestSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gost_service_request_duration_seconds",
				Help: "Distribution of request latencies",
				Buckets: []float64{
					.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 20, 30,
				},
			},
			[]string{"service"}),
		requestInputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_request_transfer_input_bytes_total",
				Help: "Total request input data transfer size in bytes",
			},
			[]string{"service"}),
		requestOutputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_request_transfer_output_bytes_total",
				Help: "Total request output data transfer size in bytes",
			},
			[]string{"service"}),
	}
	prometheus.MustRegister(m.services)
	prometheus.MustRegister(m.requests)
	prometheus.MustRegister(m.requestsInFlight)
	prometheus.MustRegister(m.requestSeconds)
	prometheus.MustRegister(m.requestInputBytes)
	prometheus.MustRegister(m.requestOutputBytes)
	return m
}

func Services() prometheus.Gauge {
	return global.services
}

func Requests(service string) prometheus.Counter {
	return global.requests.With(prometheus.Labels{"service": service})
}

func RequestsInFlight(service string) prometheus.Gauge {
	return global.requestsInFlight.With(prometheus.Labels{"service": service})
}

func RequestSeconds(service string) prometheus.Observer {
	return global.requestSeconds.With(prometheus.Labels{"service": service})
}

func RequestInputBytes(service string) prometheus.Counter {
	return global.requestInputBytes.With(prometheus.Labels{"service": service})
}

func RequestOutputBytes(service string) prometheus.Counter {
	return global.requestOutputBytes.With(prometheus.Labels{"service": service})
}
