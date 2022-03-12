package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metrics = newMetrics()
)

type Gauge interface {
	Inc()
	Dec()
	Add(float64)
	Set(float64)
}

type Counter interface {
	Inc()
	Add(float64)
}

type Observer interface {
	Observe(float64)
}

type Metrics struct {
	services         prometheus.Gauge
	requests         *prometheus.CounterVec
	requestsInFlight *prometheus.GaugeVec
	requestSeconds   *prometheus.HistogramVec
	inputBytes       *prometheus.CounterVec
	outputBytes      *prometheus.CounterVec
	handlerErrors    *prometheus.CounterVec
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
					.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 30, 60,
				},
			},
			[]string{"service"}),
		inputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_transfer_input_bytes_total",
				Help: "Total service input data transfer size in bytes",
			},
			[]string{"service"}),
		outputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_transfer_output_bytes_total",
				Help: "Total service output data transfer size in bytes",
			},
			[]string{"service"}),
		handlerErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_handler_errors_total",
				Help: "Total service handler errors",
			},
			[]string{"service"}),
	}
	prometheus.MustRegister(m.services)
	prometheus.MustRegister(m.requests)
	prometheus.MustRegister(m.requestsInFlight)
	prometheus.MustRegister(m.requestSeconds)
	prometheus.MustRegister(m.inputBytes)
	prometheus.MustRegister(m.outputBytes)
	prometheus.MustRegister(m.handlerErrors)
	return m
}

func Services() Gauge {
	return metrics.services
}

func Requests(service string) Counter {
	return metrics.requests.With(prometheus.Labels{"service": service})
}

func RequestsInFlight(service string) Gauge {
	return metrics.requestsInFlight.With(prometheus.Labels{"service": service})
}

func RequestSeconds(service string) Observer {
	return metrics.requestSeconds.With(prometheus.Labels{"service": service})
}

func InputBytes(service string) Counter {
	return metrics.inputBytes.With(prometheus.Labels{"service": service})
}

func OutputBytes(service string) Counter {
	return metrics.outputBytes.With(prometheus.Labels{"service": service})
}

func HandlerErrors(service string) Counter {
	return metrics.handlerErrors.With(prometheus.Labels{"service": service})
}
