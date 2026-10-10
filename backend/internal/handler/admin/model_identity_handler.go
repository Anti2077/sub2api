package admin

import (
	"database/sql"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type ModelIdentityHandler struct{ svc *service.ModelIdentityService }

func (h *ModelIdentityHandler) Service() *service.ModelIdentityService { return h.svc }

func NewModelIdentityHandler(svc *service.ModelIdentityService) *ModelIdentityHandler {
	return &ModelIdentityHandler{svc: svc}
}
func id(c *gin.Context) (int64, bool) {
	v, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return v, true
}
func (h *ModelIdentityHandler) Models(c *gin.Context) {
	models, err := h.svc.Models(c.Request.Context())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"engine_commit": h.svc.EngineCommit(), "models": models})
}

func (h *ModelIdentityHandler) Settings(c *gin.Context) {
	settings, err := h.svc.Settings(c.Request.Context())
	if err != nil {
		response.InternalError(c, "could not read model identity settings")
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *ModelIdentityHandler) SaveSettings(c *gin.Context) {
	var req service.IdentitySettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid model identity settings")
		return
	}
	settings, err := h.svc.SaveSettings(c.Request.Context(), req.PublicBaseURL)
	if err != nil {
		response.BadRequest(c, "Could not save Base URL; use a public HTTPS URL without credentials, query parameters or fragments")
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *ModelIdentityHandler) PlannedAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.svc.PlannedAccounts(c.Request.Context(), page, size, c.Query("search"))
	if err != nil {
		response.InternalError(c, "could not read detection accounts")
		return
	}
	c.JSON(http.StatusOK, result)
}
func (h *ModelIdentityHandler) Config(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	x, e := h.svc.Config(c, v)
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			response.NotFound(c, "identity configuration not found")
		} else {
			response.InternalError(c, "could not read identity configuration")
		}
		return
	}
	c.JSON(http.StatusOK, x)
}

type identityConfigReq struct {
	UserID  int64 `json:"user_id" binding:"required"`
	GroupID int64 `json:"group_id" binding:"required"`
}

func (h *ModelIdentityHandler) SaveConfig(c *gin.Context) {
	account, ok := id(c)
	if !ok {
		return
	}
	var req identityConfigReq
	if e := c.ShouldBindJSON(&req); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	x, e := h.svc.Configure(c, account, req.UserID, req.GroupID)
	if e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.JSON(http.StatusOK, x)
}
func (h *ModelIdentityHandler) Plans(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	x, e := h.svc.Plans(c, v)
	if e != nil {
		response.InternalError(c, e.Error())
		return
	}
	c.JSON(http.StatusOK, x)
}

type identityPlanReq struct {
	AccountID       int64  `json:"account_id"`
	RequestModel    string `json:"request_model"`
	ExpectedModel   string `json:"expected_model"`
	IntervalMinutes int    `json:"interval_minutes"`
	Enabled         bool   `json:"enabled"`
}

func (h *ModelIdentityHandler) SavePlan(c *gin.Context) {
	var req identityPlanReq
	if e := c.ShouldBindJSON(&req); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	p := &service.IdentityPlan{ID: 0, AccountID: req.AccountID, RequestModel: req.RequestModel, ExpectedModel: req.ExpectedModel, IntervalMinutes: req.IntervalMinutes, Enabled: req.Enabled}
	if e := h.svc.SavePlan(c, p); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.JSON(http.StatusOK, p)
}
func (h *ModelIdentityHandler) UpdatePlan(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	var req identityPlanReq
	if e := c.ShouldBindJSON(&req); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	p := &service.IdentityPlan{ID: v, AccountID: req.AccountID, RequestModel: req.RequestModel, ExpectedModel: req.ExpectedModel, IntervalMinutes: req.IntervalMinutes, Enabled: req.Enabled}
	if e := h.svc.SavePlan(c, p); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.JSON(http.StatusOK, p)
}
func (h *ModelIdentityHandler) DeletePlan(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	if e := h.svc.DeletePlan(c, v); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *ModelIdentityHandler) Run(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	x, e := h.svc.Enqueue(c, v)
	if e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.JSON(http.StatusAccepted, x)
}
func (h *ModelIdentityHandler) Cancel(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	if e := h.svc.Cancel(c, v); e != nil {
		response.BadRequest(c, e.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *ModelIdentityHandler) RunStatus(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	x, e := h.svc.Run(c, v)
	if e != nil {
		response.NotFound(c, "run not found")
		return
	}
	c.JSON(http.StatusOK, x)
}
func (h *ModelIdentityHandler) History(c *gin.Context) {
	v, ok := id(c)
	if !ok {
		return
	}
	x, e := h.svc.History(c, v)
	if e != nil {
		response.InternalError(c, e.Error())
		return
	}
	c.JSON(http.StatusOK, x)
}
