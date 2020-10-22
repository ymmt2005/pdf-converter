package sub

import "github.com/prometheus/client_golang/prometheus"

const (
	metricNS         = "pdf_converter"
	conversionSystem = "conversion"
)

var (
	requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNS,
		Name:      "requests_total",
		Help:      "the total number of HTTP requests",
	}, []string{"status"})

	conversionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNS,
		Subsystem: conversionSystem,
		Name:      "total",
		Help:      "the total number of conversions",
	}, []string{"extension"})

	conversionFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNS,
		Subsystem: conversionSystem,
		Name:      "failed",
		Help:      "the number of failed conversions",
	}, []string{"extension"})

	conversionSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNS,
		Subsystem: conversionSystem,
		Name:      "duration_seconds",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 11),
		Help:      "histogram of latencies for PDF conversion",
	}, []string{"extension"})

	sourceBytes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNS,
		Subsystem: conversionSystem,
		Name:      "source_bytes",
		Buckets:   prometheus.ExponentialBuckets(1<<20, 2, 11),
		Help:      "histogram of source data length for PDF conversion",
	}, []string{"extension"})

	outputBytes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNS,
		Subsystem: conversionSystem,
		Name:      "output_bytes",
		Buckets:   prometheus.ExponentialBuckets(1<<20, 2, 11),
		Help:      "histogram of generated data length for PDF conversion",
	}, []string{"extension"})
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(conversionTotal)
	prometheus.MustRegister(conversionFailed)
	prometheus.MustRegister(conversionSeconds)
	prometheus.MustRegister(sourceBytes)
	prometheus.MustRegister(outputBytes)
}
