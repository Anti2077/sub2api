package service

import (
	"strings"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const AccountTagsExtraKey = "account_tags"
const MaxAccountTags = 32
const MaxAccountTagLength = 128

type AccountTag struct {
	Name         string `json:"name"`
	AccountCount int64  `json:"account_count"`
}

// NormalizeAccountTags keeps labels case-sensitive, trims edges and preserves order.
func NormalizeAccountTags(tags []string) ([]string, error) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]bool)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !utf8.ValidString(tag) || utf8.RuneCountInString(tag) > MaxAccountTagLength || strings.ContainsFunc(tag, unicode.IsControl) {
			return nil, infraerrors.BadRequest("INVALID_ACCOUNT_TAGS", "Each tag must contain 1–128 characters without control characters")
		}
		if !seen[tag] {
			result = append(result, tag)
			seen[tag] = true
		}
	}
	if len(result) > MaxAccountTags {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_TAGS", "An account supports at most 32 tags")
	}
	return result, nil
}

// NormalizeAccountTagsExtra also validates requests arriving through backup,
// OAuth, incremental-extra and bulk-update paths.
func NormalizeAccountTagsExtra(extra map[string]any) error {
	raw, exists := extra[AccountTagsExtraKey]
	if !exists {
		return nil
	}
	var tags []string
	switch values := raw.(type) {
	case []string:
		tags = values
	case []any:
		for _, value := range values {
			tag, ok := value.(string)
			if !ok {
				return infraerrors.BadRequest("INVALID_ACCOUNT_TAGS", "Tags must be an array of strings")
			}
			tags = append(tags, tag)
		}
	default:
		return infraerrors.BadRequest("INVALID_ACCOUNT_TAGS", "Tags must be an array of strings")
	}
	normalized, err := NormalizeAccountTags(tags)
	if err != nil {
		return err
	}
	extra[AccountTagsExtraKey] = normalized
	return nil
}
