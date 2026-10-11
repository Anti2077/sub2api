package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
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
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid folder")
		return
	}
	result, err := h.accountFolders.Save(c.Request.Context(), req.ID, req.Name, req.ParentID)
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
func (h *AccountHandler) MoveToFolder(c *gin.Context) {
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
		FolderID   *int64  `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid folder move")
		return
	}
	count, err := h.accountFolders.Move(c.Request.Context(), req.AccountIDs, req.FolderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"moved": count})
}
