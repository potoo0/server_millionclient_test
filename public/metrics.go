package public

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var addr = ":8112"

var (
	ConnectionCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "connections",
		Help: "The total number of connections",
	})
	RequestCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ops",
		Help: "The total number of processed events",
	})
	Latency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "latency",
		Buckets: []float64{10, 50, 100, 200, 500, 1000},
	})
)

func ServeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	Logger.Info("Serving metrics at /metrics", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}
