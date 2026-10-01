package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyDTOIncludesGroupModelAllowlist(t *testing.T) {
	group := &service.Group{ID: 1, ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"alias", "gpt-*"}}}
	encoded, err := json.Marshal(APIKeyFromService(&service.APIKey{Group: group}))
	require.NoError(t, err)
	var result struct {
		Group struct {
			ModelAllowlist service.GroupModelAllowlist `json:"model_allowlist"`
		} `json:"group"`
	}
	require.NoError(t, json.Unmarshal(encoded, &result))
	require.Equal(t, group.ModelAllowlist, result.Group.ModelAllowlist)
	admin := GroupFromServiceAdmin(group)
	encoded, err = json.Marshal(admin)
	require.NoError(t, err)
	var adminResult struct {
		ModelAllowlist service.GroupModelAllowlist `json:"model_allowlist"`
	}
	require.NoError(t, json.Unmarshal(encoded, &adminResult))
	require.Equal(t, group.ModelAllowlist, adminResult.ModelAllowlist)
}
