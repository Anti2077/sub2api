package admin

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetRoutingMonitorSnapshot returns active requests and the last minute
// of completed/failed routing events.
// GET /api/v1/admin/ops/routing-monitor/snapshot
func (h *OpsHandler) GetRoutingMonitorSnapshot(c *gin.Context) {
	if h == nil || h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	monitor := h.opsService.RoutingMonitor()
	if monitor == nil {
		response.Success(c, map[string]any{
			"generated_at": time.Now().UTC(),
			"active":       []any{},
			"recent":       []any{},
		})
		return
	}
	snapshot, err := monitor.Snapshot(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Failed to load routing monitor snapshot")
		return
	}
	response.Success(c, snapshot)
}
