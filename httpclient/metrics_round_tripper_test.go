/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsRoundTripper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	collector := NewPrometheusMetricsCollector("")
	defer collector.Unregister()

	metricsRoundTripper := NewMetricsRoundTripperWithOpts(http.DefaultTransport, collector, MetricsRoundTripperOpts{
		ClientType: "test-client-type",
	})
	client := &http.Client{Transport: metricsRoundTripper}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, nil)
	require.NoError(t, err)

	r, err := client.Do(req)
	defer func() { _ = r.Body.Close() }()
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 1)
	go func() {
		collector.Durations.Collect(ch)
		close(ch)
	}()

	var metricCount int
	for range ch {
		metricCount++
	}

	require.Equal(t, metricCount, 1)
}

func TestNewMetricsCollectionRequiredRoundTripper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	metricsRoundTripper := NewMetricsRoundTripper(http.DefaultTransport, nil)
	client := &http.Client{Transport: metricsRoundTripper}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, nil)
	require.NoError(t, err)

	_, err = client.Do(req) // nolint:bodyclose
	require.Error(t, err)
	require.Contains(t, err.Error(), "metrics collector is not provided")
}
