package service

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type AccountFolder struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ParentID     *int64 `json:"parent_id"`
	AccountCount int64  `json:"account_count"`
	ErrorCount   int64  `json:"error_count"`
}
type AccountFolderOverview struct {
	Folders []AccountFolder `json:"folders"`
	Total   int64           `json:"total"`
	Unfiled int64           `json:"unfiled"`
}
type AccountFolderRepository interface {
	Overview(context.Context) (*AccountFolderOverview, error)
	Save(context.Context, int64, string, *int64) (*AccountFolder, error)
	Delete(context.Context, int64) error
	Move(context.Context, []int64, *int64) (int64, error)
}
type AccountFolderService struct{ repo AccountFolderRepository }

func NewAccountFolderService(repo AccountFolderRepository) *AccountFolderService {
	return &AccountFolderService{repo: repo}
}
func (s *AccountFolderService) Overview(ctx context.Context) (*AccountFolderOverview, error) {
	return s.repo.Overview(ctx)
}
func (s *AccountFolderService) Save(ctx context.Context, id int64, name string, parent *int64) (*AccountFolder, error) {
	name = strings.TrimSpace(name)
	if id < 0 || name == "" || utf8.RuneCountInString(name) > 80 || (parent != nil && (*parent <= 0 || *parent == id)) {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_FOLDER", "Folder name must contain 1–80 characters and parent must be valid")
	}
	return s.repo.Save(ctx, id, name, parent)
}
func (s *AccountFolderService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
func (s *AccountFolderService) Move(ctx context.Context, ids []int64, folder *int64) (int64, error) {
	if len(ids) == 0 || len(ids) > 10000 || (folder != nil && *folder <= 0) {
		return 0, infraerrors.BadRequest("INVALID_ACCOUNT_FOLDER_MOVE", "Select 1–10000 accounts and a valid folder")
	}
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			return 0, infraerrors.BadRequest("INVALID_ACCOUNT_FOLDER_MOVE", "Invalid account ID")
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return s.repo.Move(ctx, unique, folder)
}

type accountFolderFilterKey struct{}

// WithAccountFolderFilter carries an admin-only organizational scope through all
// list paths, including pagination, scheduler-score pools and filtered bulk edits.
// -1 means unfiled; a positive ID includes that folder's direct children.
func WithAccountFolderFilter(ctx context.Context, value string) (context.Context, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ctx, nil
	}
	id := int64(-1)
	if value != "unfiled" {
		var err error
		id, err = strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, infraerrors.BadRequest("INVALID_ACCOUNT_FOLDER_FILTER", "Invalid folder filter")
		}
	}
	return context.WithValue(ctx, accountFolderFilterKey{}, id), nil
}
func AccountFolderFilterFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(accountFolderFilterKey{}).(int64)
	return id, ok
}
