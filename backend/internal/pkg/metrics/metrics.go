// Package metrics exposes a small, dependency-free Prometheus endpoint. The
// application is intentionally a single-process modular monolith at this stage,
// so process-local counters are sufficient; a multi-instance deployment should
// add a shared store or scrape every instance.
package metrics

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fixphone/backend/internal/pkg/version"

	"github.com/gin-gonic/gin"
)

var buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestKey struct {
	method string
	route  string
	status int
}

type durationKey struct {
	method string
	route  string
}

type durationSeries struct {
	count   uint64
	sum     float64
	buckets []uint64
}

var (
	mu       sync.Mutex
	requests = make(map[requestKey]uint64)
	duration = make(map[durationKey]*durationSeries)
	started  = time.Now()
)

// Observe records one finished request. route should be a bounded value such as
// c.FullPath(); unknown routes are collapsed by the caller.
func Observe(method, route string, status int, elapsed time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	seconds := elapsed.Seconds()

	mu.Lock()
	defer mu.Unlock()
	requests[requestKey{method: method, route: route, status: status}]++

	key := durationKey{method: method, route: route}
	series := duration[key]
	if series == nil {
		series = &durationSeries{buckets: make([]uint64, len(buckets))}
		duration[key] = series
	}
	series.count++
	series.sum += seconds
	for i, bound := range buckets {
		if seconds <= bound {
			series.buckets[i]++
		}
	}
}

// Middleware records latency and status for every request.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		Observe(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}

// Handler renders the Prometheus text exposition format. The optional token is
// checked by MetricsAuth before this handler runs. probe may be nil.
func Handler(probe func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		write(c.Writer, probe, c.Request.Context())
	}
}

func write(w io.Writer, probe func(context.Context) error, ctx context.Context) {
	dbUp := 1
	if probe != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := probe(probeCtx); err != nil {
			dbUp = 0
		}
	}

	mu.Lock()
	requestSnapshot := make(map[requestKey]uint64, len(requests))
	for key, value := range requests {
		requestSnapshot[key] = value
	}
	durationSnapshot := make(map[durationKey]durationSeries, len(duration))
	for key, value := range duration {
		copied := durationSeries{count: value.count, sum: value.sum, buckets: append([]uint64(nil), value.buckets...)}
		durationSnapshot[key] = copied
	}
	mu.Unlock()

	fmt.Fprintln(w, "# HELP fixphone_build_info Build metadata.")
	fmt.Fprintln(w, "# TYPE fixphone_build_info gauge")
	fmt.Fprintf(w, "fixphone_build_info{version=%q,commit=%q} 1\n", version.Version, version.Commit)

	fmt.Fprintln(w, "# HELP fixphone_uptime_seconds Process uptime in seconds.")
	fmt.Fprintln(w, "# TYPE fixphone_uptime_seconds gauge")
	fmt.Fprintf(w, "fixphone_uptime_seconds %.3f\n", time.Since(started).Seconds())

	fmt.Fprintln(w, "# HELP fixphone_db_up Whether the database responded to a ping (1=up).")
	fmt.Fprintln(w, "# TYPE fixphone_db_up gauge")
	fmt.Fprintf(w, "fixphone_db_up %d\n", dbUp)

	fmt.Fprintln(w, "# HELP fixphone_go_goroutines Number of goroutines.")
	fmt.Fprintln(w, "# TYPE fixphone_go_goroutines gauge")
	fmt.Fprintf(w, "fixphone_go_goroutines %d\n", runtime.NumGoroutine())

	fmt.Fprintln(w, "# HELP fixphone_http_requests_total Finished HTTP requests.")
	fmt.Fprintln(w, "# TYPE fixphone_http_requests_total counter")
	requestKeys := make([]requestKey, 0, len(requestSnapshot))
	for key := range requestSnapshot {
		requestKeys = append(requestKeys, key)
	}
	sort.Slice(requestKeys, func(i, j int) bool {
		if requestKeys[i].route != requestKeys[j].route {
			return requestKeys[i].route < requestKeys[j].route
		}
		if requestKeys[i].method != requestKeys[j].method {
			return requestKeys[i].method < requestKeys[j].method
		}
		return requestKeys[i].status < requestKeys[j].status
	})
	for _, key := range requestKeys {
		fmt.Fprintf(w, "fixphone_http_requests_total{method=%q,route=%q,status=%q} %d\n",
			key.method, key.route, strconv.Itoa(key.status), requestSnapshot[key])
	}

	histogramKeys := make([]durationKey, 0, len(durationSnapshot))
	for key := range durationSnapshot {
		histogramKeys = append(histogramKeys, key)
	}
	sort.Slice(histogramKeys, func(i, j int) bool {
		if histogramKeys[i].route != histogramKeys[j].route {
			return histogramKeys[i].route < histogramKeys[j].route
		}
		return histogramKeys[i].method < histogramKeys[j].method
	})
	fmt.Fprintln(w, "# HELP fixphone_http_request_duration_seconds HTTP request latency.")
	fmt.Fprintln(w, "# TYPE fixphone_http_request_duration_seconds histogram")
	for _, key := range histogramKeys {
		series := durationSnapshot[key]
		for i, bound := range buckets {
			fmt.Fprintf(w, "fixphone_http_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %d\n",
				key.method, key.route, trimFloat(bound), series.buckets[i])
		}
		fmt.Fprintf(w, "fixphone_http_request_duration_seconds_bucket{method=%q,route=%q,le=\"+Inf\"} %d\n",
			key.method, key.route, series.count)
		fmt.Fprintf(w, "fixphone_http_request_duration_seconds_sum{method=%q,route=%q} %s\n",
			key.method, key.route, trimFloat(series.sum))
		fmt.Fprintf(w, "fixphone_http_request_duration_seconds_count{method=%q,route=%q} %d\n",
			key.method, key.route, series.count)
	}
}

// MetricsAuth protects the endpoint when a token is configured. An empty token
// leaves the endpoint open; production exposes it only on the internal network.
func MetricsAuth(token string) gin.HandlerFunc {
	expected := strings.TrimSpace(token)
	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if provided == "" {
			provided = strings.TrimSpace(c.GetHeader("X-Metrics-Token"))
		}
		if provided != expected {
			c.AbortWithStatusJSON(404, gin.H{"success": false, "error": gin.H{"code": "NOT_FOUND", "message": "Endpoint tidak ditemukan."}})
			return
		}
		c.Next()
	}
}

func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
