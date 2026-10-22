package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type smartFolderHandlerRepo struct {
	service.AccountFolderRepository
	rules     *service.AccountFolderRules
	ids       []int64
	tags      []string
	operation string
}

func (r *smartFolderHandlerRepo) Save(_ context.Context, id int64, name string, parent *int64, rules *service.AccountFolderRules) (*service.AccountFolder, error) {
	r.rules = rules
	return &service.AccountFolder{ID: 1, Name: name, ParentID: parent, Rules: rules}, nil
}
func (r *smartFolderHandlerRepo) Tags(context.Context) ([]service.AccountTag, error) {
	return []service.AccountTag{{Name: "team A", AccountCount: 2}}, nil
}
func (r *smartFolderHandlerRepo) UpdateTags(_ context.Context, ids []int64, tags []string, operation string) (int64, error) {
	r.ids, r.tags, r.operation = ids, tags, operation
	return int64(len(ids)), nil
}
func TestAccountSmartFolderHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &smartFolderHandlerRepo{}
	h := &AccountHandler{accountFolders: service.NewAccountFolderService(repo)}
	router := gin.New()
	router.POST("/folders", h.SaveFolder)
	router.GET("/tags", h.AccountTags)
	router.POST("/tags", h.UpdateAccountTags)
	router.POST("/folders/move", h.MoveToFolder)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}
	for _, body := range []string{`{"name":"Empty"}`, `{"name":"Empty","rules":{}}`, `{"name":"Bad","rules":{"tags":"label"}}`, `{"name":"Bad","rules":{"tags":["x"],"tag_state":"untagged"}}`} {
		require.Equal(t, http.StatusBadRequest, request(http.MethodPost, "/folders", body).Code)
	}
	w := request(http.MethodPost, "/folders", `{"name":"Disabled OpenAI","rules":{"platforms":["openai"],"schedulable":false}}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.rules.Schedulable)
	require.False(t, *repo.rules.Schedulable)
	w = request(http.MethodPost, "/tags", `{"account_ids":[10,10,20],"tags":[" team A ","team A"],"operation":"remove"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []int64{10, 20}, repo.ids)
	require.Equal(t, []string{"team A"}, repo.tags)
	require.Equal(t, "remove", repo.operation)
	for _, body := range []string{`{"account_ids":[],"tags":["x"],"operation":"add"}`, `{"account_ids":[1],"tags":[1],"operation":"add"}`, `{"account_ids":[1],"tags":["x"],"operation":"replace"}`} {
		require.Equal(t, http.StatusBadRequest, request(http.MethodPost, "/tags", body).Code)
	}
	w = request(http.MethodGet, "/tags", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "team A")
	require.Equal(t, http.StatusBadRequest, request(http.MethodPost, "/folders/move", `{"account_ids":[1],"folder_id":1}`).Code)
}
