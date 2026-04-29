package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type Registry struct {
	RequestsTotal    uint64
	ErrorsTotal      uint64
	ConfigUpdates    uint64
	ActiveRequests   int64
	StartTime        time.Time
}

var Global = &Registry{
	StartTime: time.Now(),
}

func (r *Registry) IncRequests() { atomic.AddUint64(&r.RequestsTotal, 1) }
func (r *Registry) IncErrors()   { atomic.AddUint64(&r.ErrorsTotal, 1) }
func (r *Registry) IncUpdates()  { atomic.AddUint64(&r.ConfigUpdates, 1) }
func (r *Registry) AddActive(n int64) { atomic.AddInt64(&r.ActiveRequests, n) }

func (r *Registry) WritePrometheus(w http.ResponseWriter) {
	fmt.Fprintf(w, "# HELP nautilus_requests_total Total number of processed requests\n")
	fmt.Fprintf(w, "# TYPE nautilus_requests_total counter\n")
	fmt.Fprintf(w, "nautilus_requests_total %d\n", atomic.LoadUint64(&r.RequestsTotal))

	fmt.Fprintf(w, "# HELP nautilus_errors_total Total number of failed requests\n")
	fmt.Fprintf(w, "# TYPE nautilus_errors_total counter\n")
	fmt.Fprintf(w, "nautilus_errors_total %d\n", atomic.LoadUint64(&r.ErrorsTotal))

	fmt.Fprintf(w, "# HELP nautilus_config_updates_total Total number of configuration swaps\n")
	fmt.Fprintf(w, "# TYPE nautilus_config_updates_total counter\n")
	fmt.Fprintf(w, "nautilus_config_updates_total %d\n", atomic.LoadUint64(&r.ConfigUpdates))

	fmt.Fprintf(w, "# HELP nautilus_active_requests Number of requests currently being processed\n")
	fmt.Fprintf(w, "# TYPE nautilus_active_requests gauge\n")
	fmt.Fprintf(w, "nautilus_active_requests %d\n", atomic.LoadInt64(&r.ActiveRequests))

	fmt.Fprintf(w, "# HELP nautilus_uptime_seconds Engine uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE nautilus_uptime_seconds gauge\n")
	fmt.Fprintf(w, "nautilus_uptime_seconds %.0f\n", time.Since(r.StartTime).Seconds())
}
