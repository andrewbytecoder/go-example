package exporter

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/go-example/options"
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// handler wraps an unfiltered http.Handler but uses a filtered handler,
// created on the fly, if filtering is requested. Create instances with
// newHandler.
type handler struct {
	handler http.Handler

	includeExporterMetrics bool
	// exporterMetricsRegistry is a separate registry for the metrics about
	// the exporter itself.
	exporterMetricsRegistry *prometheus.Registry

	maxRequests  int
	metricsPaths []string
	logger       *zap.Logger
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
	return
}

// innerHandler is used to create both the one unfiltered http.Handler to be
// wrapped by the outer handler and also the filtered handlers created on the
// fly. The former is accomplished by calling innerHandler without any arguments
// (in which case it will log all the collectors enabled via command-line
// flags).
func (h *handler) innerHandler() (http.Handler, error) {
	nc, err := NewNodeCollector(h.logger)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}

	r := prometheus.NewRegistry()
	if err := r.Register(nc); err != nil {
		return nil, fmt.Errorf("couldn't register node collector: %s", err)
	}

	var handler http.Handler
	if h.includeExporterMetrics {
		handler = promhttp.HandlerFor(
			prometheus.Gatherers{h.exporterMetricsRegistry, r},
			promhttp.HandlerOpts{
				ErrorHandling:       promhttp.ContinueOnError,
				MaxRequestsInFlight: h.maxRequests,
				Registry:            h.exporterMetricsRegistry,
			},
		)
		// Note that we have to use h.exporterMetricsRegistry here to
		// use the same promhttp metrics for all expositions.
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	} else {
		handler = promhttp.HandlerFor(
			r,
			promhttp.HandlerOpts{
				ErrorHandling:       promhttp.ContinueOnError,
				MaxRequestsInFlight: h.maxRequests,
			},
		)
	}

	return handler, nil
}

// SetMetricsPath 设置 metricsPath
func SetMetricsPath(metricsPath string) options.Option {
	return func(c interface{}) {
		c.(*handler).metricsPaths = append(c.(*handler).metricsPaths, metricsPath)
	}
}

// SetMaxRequests 配置最大请求数
func SetMaxRequests(maxRequests int) options.Option {
	return func(c interface{}) {
		c.(*handler).maxRequests = maxRequests
	}
}

// SetIncludeExporterMetrics 配置是否包含 exporter 监控
func SetIncludeExporterMetrics(includeExporterMetrics bool) options.Option {
	return func(c interface{}) {
		c.(*handler).includeExporterMetrics = includeExporterMetrics
	}
}

// SetLogger 配置日志
func SetLogger(logger *zap.Logger) options.Option {
	return func(c interface{}) {
		c.(*handler).logger = logger
	}
}

func Start(engine *gin.Engine, opts ...options.Option) error {
	h := &handler{}
	for _, opt := range opts {
		opt(h)
	}

	// 如果需要包含 exporter 监控，创建exporter监控器
	if h.includeExporterMetrics {
		h.exporterMetricsRegistry = prometheus.NewRegistry()
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	if innerHandler, err := h.innerHandler(); err != nil {
		h.logger.Error("Couldn't create metrics handler", zap.Error(err))
		return err
	} else {
		h.handler = innerHandler
	}

	metricsGroup := engine.Group("")
	for _, path := range h.metricsPaths {
		metricsGroup.GET(path, gin.WrapH(h))
	}
	return nil
}
