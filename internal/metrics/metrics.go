package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	prometheusRegistry *prometheus.Registry
	prometheusHandler  http.Handler
	panicTotal         prometheus.Counter
	requestCount       *prometheus.CounterVec
	statusCount        *prometheus.CounterVec
}

var m *metrics

func init() {
	m = &metrics{
		prometheusRegistry: prometheus.NewRegistry(),
	}

	m.prometheusRegistry.Register(collectors.NewGoCollector())
	m.prometheusRegistry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	m.panicTotal = promauto.With(m.prometheusRegistry).NewCounter(prometheus.CounterOpts{
		Name: "http_server_panics_recovered_total",
		Help: "Total number of requests recovered after an internal panic.",
	})

	m.requestCount = promauto.With(m.prometheusRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_server_requests_path_count",
			Help: "Application request path count",
		},
		[]string{"method", "uri", "status"},
	)

	m.statusCount = promauto.With(m.prometheusRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_server_requests_status_count",
			Help: "Application request status count",
		},
		[]string{"status"},
	)

	m.prometheusHandler = promhttp.HandlerFor(
		m.prometheusRegistry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		})
}

func Mertics() *metrics {
	return m
}

func (m *metrics) PanicInc() {
	m.panicTotal.Inc()
}

func (m *metrics) RequestInc(status int) {
	m.statusCount.WithLabelValues(strconv.Itoa(status)).Inc()
}
func (m *metrics) RequestStatusInc(method, uri string, status int) {
	m.requestCount.WithLabelValues(method, uri, strconv.Itoa(status)).Inc()
}

func (m *metrics) Handler() http.Handler {
	return m.prometheusHandler
}
