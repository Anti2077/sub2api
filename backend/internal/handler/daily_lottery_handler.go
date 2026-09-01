package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type DailyLotteryHandler struct {
	service *service.DailyLotteryService
}

func NewDailyLotteryHandler(dailyLotteryService *service.DailyLotteryService) *DailyLotteryHandler {
	return &DailyLotteryHandler{service: dailyLotteryService}
}

func (h *DailyLotteryHandler) Status(c *gin.Context) {
	userID, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	status, err := h.service.GetStatus(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *DailyLotteryHandler) CheckIn(c *gin.Context) {
	userID, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	status, err := h.service.CheckIn(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *DailyLotteryHandler) Draw(c *gin.Context) {
	userID, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	result, err := h.service.Draw(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *DailyLotteryHandler) History(c *gin.Context) {
	userID, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	entries, err := h.service.ListUserHistory(c.Request.Context(), userID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entries)
}

func dailyLotteryUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}
