package telemetry

// Exposed Prometheus metrics:
//   aleph_http_requests_total{method, path, status_code}
//   aleph_request_duration_seconds{method, path, status_code}
//   aleph_nlp_requests_total{method, status}
//   aleph_db_query_duration_seconds{operation}
//   aleph_paora_cycle_total{phase, outcome}
//   aleph_db_connections_active (gauge)

import (
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
)

var (
	NLPRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aleph_nlp_requests_total",
			Help: "Total NLP sidecar requests",
		},
		[]string{"method", "status"},
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aleph_db_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	PAORACycleTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aleph_paora_cycle_total",
			Help: "PAORA decision-engine cycle phases",
		},
		[]string{"phase", "outcome"},
	)
)

func init() {
	for _, c := range []prometheus.Collector{NLPRequestCount, DBQueryDuration, PAORACycleTotal} {
		if err := prometheus.Register(c); err != nil {
			slog.Warn("prometheus metric registration skipped", "error", err)
		}
	}
}

func RecordNLPRequest(method, status string) {
	NLPRequestCount.WithLabelValues(method, status).Inc()
}

func RecordDBQuery(operation string, durationSeconds float64) {
	DBQueryDuration.WithLabelValues(operation).Observe(durationSeconds)
}

func RecordPAORACycle(phase, outcome string) {
	PAORACycleTotal.WithLabelValues(phase, outcome).Inc()
}

func SetDBConnections(n float64) {
	promDBConnections.Set(n)
}

