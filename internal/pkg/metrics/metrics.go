// Package metrics exposes lightweight liveness and counters over HTTP so a
// long-running mynews daemon can be monitored.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// Metrics holds process counters updated from the run loop and served over HTTP.
type Metrics struct {
	feedsParsed     atomic.Int64
	parseErrors     atomic.Int64
	cyclesCompleted atomic.Int64
	lastCycleUnix   atomic.Int64

	// staleAfter is how long without a completed cycle marks the loop unhealthy.
	staleAfter time.Duration
}

// New returns a Metrics whose health check reports unhealthy when no cycle has
// completed within staleAfter.
func New(staleAfter time.Duration) *Metrics {
	return &Metrics{staleAfter: staleAfter} //nolint:exhaustruct // atomics zero-initialize
}

// FeedParsed records a successfully parsed feed.
func (m *Metrics) FeedParsed() { m.feedsParsed.Add(1) }

// ParseError records a feed that failed to parse.
func (m *Metrics) ParseError() { m.parseErrors.Add(1) }

// CycleCompleted records the end of a full parse cycle.
func (m *Metrics) CycleCompleted() {
	m.cyclesCompleted.Add(1)
	m.lastCycleUnix.Store(time.Now().Unix())
}

// Serve runs the metrics HTTP server until ctx is canceled.
func (m *Metrics) Serve(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", m.handleHealth)
	mux.HandleFunc("/metrics", m.handleMetrics)

	//nolint:exhaustruct // only addr, handler and the header timeout matter here
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: readHeaderTimeout}

	serverErr := make(chan error, 1)

	go func() { serverErr <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		// A fresh context is required: the run context is already canceled, but
		// we still want a brief grace period to drain in-flight requests.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		//nolint:contextcheck // intentionally detached from the canceled run context
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			return fmt.Errorf("shutting down metrics server: %w", err)
		}

		return nil
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("metrics server: %w", err)
	}
}

func (m *Metrics) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	lastCycle := m.lastCycleUnix.Load()

	healthy := lastCycle == 0 || time.Since(time.Unix(lastCycle, 0)) <= m.staleAfter
	if !healthy {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}

	_, _ = fmt.Fprintf(writer, "last_cycle_unix %d\n", lastCycle)
}

func (m *Metrics) handleMetrics(writer http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(writer,
		"mynews_feeds_parsed_total %d\n"+
			"mynews_parse_errors_total %d\n"+
			"mynews_cycles_completed_total %d\n"+
			"mynews_last_cycle_timestamp_seconds %d\n",
		m.feedsParsed.Load(),
		m.parseErrors.Load(),
		m.cyclesCompleted.Load(),
		m.lastCycleUnix.Load(),
	)
}
