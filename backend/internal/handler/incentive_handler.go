package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type IncentiveHandler struct{ service *service.IncentiveService }

func NewIncentiveHandler(s *service.IncentiveService) *IncentiveHandler {
	return &IncentiveHandler{service: s}
}
func (h *IncentiveHandler) Config(c *gin.Context) {
	v, e := h.service.Configs(c.Request.Context())
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *IncentiveHandler) Save(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var input service.IncentiveConfig
	if e := c.ShouldBindJSON(&input); e != nil {
		response.BadRequest(c, "Invalid configuration")
		return
	}
	v, e := h.service.Save(c.Request.Context(), input, id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *IncentiveHandler) Status(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	h.status(c, id)
}
func (h *IncentiveHandler) AdminStatus(c *gin.Context) { h.status(c, 0) }
func (h *IncentiveHandler) status(c *gin.Context, id int64) {
	v, e := h.service.Status(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *IncentiveHandler) History(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	h.history(c, id)
}
func (h *IncentiveHandler) AdminHistory(c *gin.Context) { h.history(c, 0) }
func (h *IncentiveHandler) history(c *gin.Context, id int64) {
	v, e := h.service.History(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *IncentiveHandler) Reset(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid period")
		return
	}
	if e = h.service.Reset(c.Request.Context(), id, actor); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"reset": true})
}
func (h *IncentiveHandler) Draw(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var input struct {
		RequestKey string `json:"request_key"`
	}
	if e := c.ShouldBindJSON(&input); e != nil {
		response.BadRequest(c, "Invalid draw request")
		return
	}
	v, e := h.service.Draw(c.Request.Context(), id, input.RequestKey)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
