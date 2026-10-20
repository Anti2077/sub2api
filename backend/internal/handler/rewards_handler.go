package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/gin-gonic/gin"
	"strconv"
)

type RewardsHandler struct{ Store *rewards.Store }

func NewRewardsHandler(s *rewards.Store) *RewardsHandler { return &RewardsHandler{s} }
func rewardRespond(c *gin.Context, v any, e error) {
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *RewardsHandler) Cards(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	v, e := h.Store.Cards(c.Request.Context(), id)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) AdminCards(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	v, e := h.Store.Cards(c.Request.Context(), id)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) Activate(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	e := h.Store.Activate(c.Request.Context(), user, id)
	rewardRespond(c, gin.H{"activated": e == nil}, e)
}
func (h *RewardsHandler) Grant(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v struct {
		User     int64  `json:"user_id"`
		Template int64  `json:"template_id"`
		Key      string `json:"request_key"`
		Reason   string `json:"reason"`
	}
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid grant")
		return
	}
	id, e := h.Store.Grant(c.Request.Context(), v.User, v.Template, actor, v.Key, v.Reason)
	rewardRespond(c, gin.H{"card_id": id}, e)
}
func (h *RewardsHandler) Revoke(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var v struct {
		Reason string `json:"reason"`
	}
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid reason")
		return
	}
	e := h.Store.Revoke(c.Request.Context(), id, actor, v.Reason)
	rewardRespond(c, gin.H{"revoked": e == nil}, e)
}
func (h *RewardsHandler) Templates(c *gin.Context) {
	v, e := h.Store.Templates(c.Request.Context())
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) SaveTemplate(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v rewards.Template
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid template")
		return
	}
	out, e := h.Store.SaveTemplate(c.Request.Context(), v, id)
	rewardRespond(c, out, e)
}
func (h *RewardsHandler) Config(c *gin.Context) {
	v, e := h.Store.Config(c.Request.Context(), h.Store.DB)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) SaveConfig(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v rewards.Config
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid settings")
		return
	}
	out, e := h.Store.SaveConfig(c.Request.Context(), v, id)
	rewardRespond(c, out, e)
}
func (h *RewardsHandler) Records(c *gin.Context) {
	id, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	v, e := h.Store.Records(c.Request.Context(), id, c.Query("kind"))
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) AdminRecords(c *gin.Context) {
	v, e := h.Store.Records(c.Request.Context(), 0, c.Query("kind"))
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) Farm(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	owner, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if owner == 0 {
		owner = user
	}
	v, e := h.Store.Farm(c.Request.Context(), owner, user)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) Neighbors(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	v, e := h.Store.Neighbors(c.Request.Context(), user, c.Query("search"))
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) Action(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v rewards.FarmAction
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid action")
		return
	}
	out, e := h.Store.Action(c.Request.Context(), user, v)
	rewardRespond(c, out, e)
}
func (h *RewardsHandler) Shop(c *gin.Context) {
	v, e := h.Store.Shop(c.Request.Context(), false)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) AdminShop(c *gin.Context) {
	v, e := h.Store.Shop(c.Request.Context(), true)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) SaveShop(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v rewards.Shop
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid shop item")
		return
	}
	out, e := h.Store.SaveShop(c.Request.Context(), v, user)
	rewardRespond(c, out, e)
}
func (h *RewardsHandler) SetShopPublished(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid shop ID")
		return
	}
	var v struct {
		Version   int64 `json:"version"`
		Published *bool `json:"published"`
	}
	if c.ShouldBindJSON(&v) != nil || v.Version <= 0 || v.Published == nil {
		response.BadRequest(c, "Invalid status change")
		return
	}
	out, e := h.Store.SetShopPublished(c.Request.Context(), id, v.Version, actor, *v.Published)
	rewardRespond(c, gin.H{"version": out}, e)
}
func (h *RewardsHandler) FarmLedger(c *gin.Context) {
	user, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	v, e := h.Store.FarmLedger(c.Request.Context(), user)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) AdminFarmLedger(c *gin.Context) {
	v, e := h.Store.FarmLedger(c.Request.Context(), 0)
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) Codes(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	var v struct {
		Template int64  `json:"template_id"`
		Count    int    `json:"count"`
		Days     int    `json:"days"`
		Key      string `json:"request_key"`
	}
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid code batch")
		return
	}
	out, e := h.Store.CodesWithKey(c.Request.Context(), v.Template, v.Count, v.Days, actor, v.Key)
	rewardRespond(c, gin.H{"codes": out}, e)
}
func (h *RewardsHandler) AdminCodes(c *gin.Context) {
	v, e := h.Store.AdminCodes(c.Request.Context())
	rewardRespond(c, v, e)
}
func (h *RewardsHandler) SetCodeStatus(c *gin.Context) {
	actor, ok := dailyLotteryUserID(c)
	if !ok {
		return
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid code ID")
		return
	}
	var v struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&v) != nil {
		response.BadRequest(c, "Invalid status")
		return
	}
	e = h.Store.SetCodeStatus(c.Request.Context(), id, actor, v.Status)
	rewardRespond(c, gin.H{"updated": e == nil}, e)
}
