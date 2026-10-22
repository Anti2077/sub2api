package service

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type AccountFolder struct {
	Rules        *AccountFolderRules `json:"rules"`
	ID           int64               `json:"id"`
	Name         string              `json:"name"`
	ParentID     *int64              `json:"parent_id"`
	AccountCount int64               `json:"account_count"`
	ErrorCount   int64               `json:"error_count"`
}
type AccountFolderOverview struct {
	Folders []AccountFolder `json:"folders"`
	Total   int64           `json:"total"`
	Unfiled int64           `json:"unfiled"`
}
type AccountFolderRepository interface {
	Overview(context.Context) (*AccountFolderOverview, error)
	Save(context.Context, int64, string, *int64, *AccountFolderRules) (*AccountFolder, error)
	Delete(context.Context, int64) error
	Tags(context.Context) ([]AccountTag, error)
	UpdateTags(context.Context, []int64, []string, string) (int64, error)
}
type AccountFolderService struct{ repo AccountFolderRepository }

func NewAccountFolderService(repo AccountFolderRepository) *AccountFolderService {
	return &AccountFolderService{repo: repo}
}
func (s *AccountFolderService) Overview(ctx context.Context) (*AccountFolderOverview, error) {
	return s.repo.Overview(ctx)
}
func (s *AccountFolderService) Save(ctx context.Context, id int64, name string, parent *int64, rules *AccountFolderRules) (*AccountFolder, error) {
	name = strings.TrimSpace(name)
	if id < 0 || name == "" || utf8.RuneCountInString(name) > 80 || (parent != nil && (*parent <= 0 || *parent == id)) {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_FOLDER", "Folder name must contain 1–80 characters and parent must be valid")
	}
	if err := rules.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Save(ctx, id, name, parent, rules)
}
func (s *AccountFolderService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
func (s *AccountFolderService) Tags(ctx context.Context) ([]AccountTag, error) {
	return s.repo.Tags(ctx)
}
func (s *AccountFolderService) UpdateTags(ctx context.Context, ids []int64, tags []string, operation string) (int64, error) {
	if len(ids) == 0 || len(ids) > 10000 || (operation != "add" && operation != "remove") {
		return 0, infraerrors.BadRequest("INVALID_ACCOUNT_TAG_UPDATE", "Select 1–10000 accounts and add or remove tags")
	}
	normalized, err := NormalizeAccountTags(tags)
	if err != nil {
		return 0, err
	}
	if len(normalized) == 0 {
		return 0, infraerrors.BadRequest("INVALID_ACCOUNT_TAG_UPDATE", "Select at least one tag")
	}
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			return 0, infraerrors.BadRequest("INVALID_ACCOUNT_TAG_UPDATE", "Invalid account ID")
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	return s.repo.UpdateTags(ctx, unique, normalized, operation)
}

// Different fields are combined with AND; values within a field use OR.
// TagMatch is the explicit exception: all or any of the selected labels.
type AccountFolderRules struct {
	Platforms   []string `json:"platforms,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	TagMatch    string   `json:"tag_match,omitempty"`
	TagState    string   `json:"tag_state,omitempty"`
	Name        string   `json:"name,omitempty"`
	Types       []string `json:"types,omitempty"`
	Statuses    []string `json:"statuses,omitempty"`
	Schedulable *bool    `json:"schedulable,omitempty"`
	Expiry      string   `json:"expiry,omitempty"`
}

func (r *AccountFolderRules) Validate() error {
	invalid := func() error {
		return infraerrors.BadRequest("INVALID_SMART_FOLDER_RULES", "Specify valid smart folder conditions")
	}
	if r == nil {
		return invalid()
	}
	r.Name = strings.TrimSpace(r.Name)
	if utf8.RuneCountInString(r.Name) > 200 || len(r.Platforms) > 64 || len(r.Types) > 6 || len(r.Statuses) > 3 {
		return invalid()
	}
	for _, p := range r.Platforms {
		if strings.TrimSpace(p) != p || p == "" || len(p) > 80 {
			return invalid()
		}
	}
	for _, typ := range r.Types {
		switch typ {
		case "oauth", "setup-token", "apikey", "upstream", "bedrock", "service_account":
		default:
			return invalid()
		}
	}
	for _, status := range r.Statuses {
		switch status {
		case "active", "inactive", "error":
		default:
			return invalid()
		}
	}
	if r.TagMatch != "" && r.TagMatch != "all" && r.TagMatch != "any" {
		return invalid()
	}
	if r.TagState != "" && r.TagState != "tagged" && r.TagState != "untagged" {
		return invalid()
	}
	if r.Expiry != "" && r.Expiry != "expired" && r.Expiry != "expiring_7d" && r.Expiry != "never" {
		return invalid()
	}
	tags, err := NormalizeAccountTags(r.Tags)
	if err != nil {
		return err
	}
	r.Tags = tags
	if r.TagState == "untagged" && len(tags) > 0 {
		return invalid()
	}
	if len(r.Platforms)+len(r.Tags)+len(r.Types)+len(r.Statuses) == 0 && r.Name == "" && r.TagState == "" && r.Schedulable == nil && r.Expiry == "" {
		return invalid()
	}
	return nil
}

type accountFolderFilterKey struct{}

// WithAccountFolderFilter carries an admin-only organizational scope through all
// list paths, including pagination, scheduler-score pools and filtered bulk edits.
// -1 means unmatched; a positive ID matches that folder OR its direct children.
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
