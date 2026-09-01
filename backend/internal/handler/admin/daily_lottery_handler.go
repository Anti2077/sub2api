package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type DailyLotteryHandler struct {
	service *service.DailyLotteryService
}

func NewDailyLotteryHandler(dailyLotteryService *service.DailyLotteryService) *DailyLotteryHandler {
	return &DailyLotteryHandler{service: dailyLotteryService}
}

func (h *DailyLotteryHandler) GetConfig(c *gin.Context) {
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *DailyLotteryHandler) UpdateConfig(c *gin.Context) {
	var cfg service.DailyLotteryConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	updated, err := h.service.UpdateConfig(c.Request.Context(), cfg)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *DailyLotteryHandler) History(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	entries, total, err := h.service.ListAdminHistory(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}
