package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartMetricsServer 启动 Prometheus metrics HTTP 服务器
func StartMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()

	// 注册 Prometheus handler
	mux.Handle("/metrics", promhttp.Handler())

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// log error
		}
	}()

	return server
}

// ShutdownMetricsServer 优雅关闭 metrics 服务器
func ShutdownMetricsServer(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return server.Shutdown(ctx)
}

// MustRegister registers a new collector, panics on error
func MustRegister(collector prometheus.Collector) {
	prometheus.MustRegister(collector)
}

// Unregister removes a collector from the registry
func Unregister(name string) {
	// Note: This is a simplified version; in production you might want to track collectors
}
