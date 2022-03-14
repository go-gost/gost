package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metrics *Metrics
)

func SetGlobal(m *Metrics) {
	metrics = m
}

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
	host             string
	services         *prometheus.GaugeVec
	requests         *prometheus.CounterVec
	requestsInFlight *prometheus.GaugeVec
	requestSeconds   *prometheus.HistogramVec
	inputBytes       *prometheus.CounterVec
	outputBytes      *prometheus.CounterVec
	handlerErrors    *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	host, _ := os.Hostname()
	m := &Metrics{
		host: host,
		services: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gost_services",
				Help: "Current number of services",
			},
			[]string{"host"}),

		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_requests_total",
				Help: "Total number of requests",
			},
			[]string{"host", "service"}),

		requestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gost_service_requests_in_flight",
				Help: "Current in-flight requests",
			},
			[]string{"host", "service"}),

		requestSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gost_service_request_duration_seconds",
				Help: "Distribution of request latencies",
				Buckets: []float64{
					.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 30, 60,
				},
			},
			[]string{"host", "service"}),
		inputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_transfer_input_bytes_total",
				Help: "Total service input data transfer size in bytes",
			},
			[]string{"host", "service"}),
		outputBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_transfer_output_bytes_total",
				Help: "Total service output data transfer size in bytes",
			},
			[]string{"host", "service"}),
		handlerErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gost_service_handler_errors_total",
				Help: "Total service handler errors",
			},
			[]string{"host", "service"}),
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
	if metrics == nil || metrics.services == nil {
		return nilGauge
	}
	return metrics.services.
		With(prometheus.Labels{
			"host": metrics.host,
		})
}

func Requests(service string) Counter {
	if metrics == nil || metrics.requests == nil {
		return nilCounter
	}

	return metrics.requests.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}

func RequestsInFlight(service string) Gauge {
	if metrics == nil || metrics.requestsInFlight == nil {
		return nilGauge
	}
	return metrics.requestsInFlight.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}

func RequestSeconds(service string) Observer {
	if metrics == nil || metrics.requestSeconds == nil {
		return nilObserver
	}
	return metrics.requestSeconds.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}

func InputBytes(service string) Counter {
	if metrics == nil || metrics.inputBytes == nil {
		return nilCounter
	}
	return metrics.inputBytes.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}

func OutputBytes(service string) Counter {
	if metrics == nil || metrics.outputBytes == nil {
		return nilCounter
	}
	return metrics.outputBytes.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}

func HandlerErrors(service string) Counter {
	if metrics == nil || metrics.handlerErrors == nil {
		return nilCounter
	}
	return metrics.handlerErrors.
		With(prometheus.Labels{
			"host":    metrics.host,
			"service": service,
		})
}
