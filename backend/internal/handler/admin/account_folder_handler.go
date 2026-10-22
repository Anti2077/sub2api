package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) FolderOverview(c *gin.Context) {
	result, err := h.accountFolders.Overview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) SaveFolder(c *gin.Context) {
	var req struct {
		Rules    *service.AccountFolderRules `json:"rules"`
		ID       int64                       `json:"id"`
		Name     string                      `json:"name"`
		ParentID *int64                      `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid folder")
		return
	}
	result, err := h.accountFolders.Save(c.Request.Context(), req.ID, req.Name, req.ParentID, req.Rules)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) DeleteFolder(c *gin.Context) {
	folderID, ok := id(c)
	if !ok {
		return
	}
	if err := h.accountFolders.Delete(c.Request.Context(), folderID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *AccountHandler) AccountTags(c *gin.Context) {
	result, err := h.accountFolders.Tags(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) UpdateAccountTags(c *gin.Context) {
	var req struct {
		AccountIDs []int64  `json:"account_ids"`
		Tags       []string `json:"tags"`
		Operation  string   `json:"operation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid tag update")
		return
	}
	count, err := h.accountFolders.UpdateTags(c.Request.Context(), req.AccountIDs, req.Tags, req.Operation)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": count})
}

// Older clients must not mutate obsolete manual membership records.
func (h *AccountHandler) MoveToFolder(c *gin.Context) {
	response.BadRequest(c, "Smart folders use saved conditions. Update account tags instead.")
}
