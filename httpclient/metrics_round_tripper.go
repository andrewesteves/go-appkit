/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package httpclient

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"time"
)

var (
	ClientHTTPRequestDuration *prometheus.HistogramVec
	ClassifyRequest           func(r *http.Request, reqType string) string
)

// MustInitAndRegisterMetrics initializes and registers external HTTP request duration metric.
// Panic will be raised in case of error.
func MustInitAndRegisterMetrics(namespace string) {
	ClientHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_client_request_duration_seconds",
			Help:      "A histogram of the http client requests durations.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 150, 300, 600},
		},
		[]string{"type", "remote_address", "summary", "status"},
	)
	prometheus.MustRegister(ClientHTTPRequestDuration)
}

// UnregisterMetrics unregisters external HTTP request duration metric.
func UnregisterMetrics() {
	if ClientHTTPRequestDuration != nil {
		prometheus.Unregister(ClientHTTPRequestDuration)
	}
}

// MetricsRoundTripper is an HTTP transport that measures requests done.
type MetricsRoundTripper struct {
	// Delegate is the next RoundTripper in the chain.
	Delegate http.RoundTripper

	// ReqType is a type of request.
	ReqType string
}

// NewMetricsRoundTripper creates an HTTP transport that measures requests done.
func NewMetricsRoundTripper(delegate http.RoundTripper, reqType string) http.RoundTripper {
	return &MetricsRoundTripper{Delegate: delegate, ReqType: reqType}
}

// RoundTrip measures external requests done.
func (rt *MetricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if ClientHTTPRequestDuration == nil {
		return rt.Delegate.RoundTrip(r)
	}

	status := "0"
	start := time.Now()

	resp, err := rt.Delegate.RoundTrip(r)
	if err == nil && resp != nil {
		status = strconv.Itoa(resp.StatusCode)
	}

	ClientHTTPRequestDuration.WithLabelValues(
		rt.ReqType, r.Host, requestSummary(r, rt.ReqType), status,
	).Observe(time.Since(start).Seconds())

	return resp, err
}

// requestSummary does request classification, producing non-parameterized summary for given request.
func requestSummary(r *http.Request, reqType string) string {
	if ClassifyRequest != nil {
		return ClassifyRequest(r, reqType)
	}

	return fmt.Sprintf("%s %s", r.Method, reqType)
}
