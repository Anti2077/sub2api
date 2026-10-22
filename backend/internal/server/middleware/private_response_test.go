package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPrivateNoStoreIncludesErrorsAndPreservesVary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, status := range []int{200, 401, 403, 429, 500} {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Header("Vary", "Origin")
			c.Next()
		})
		router.Use(PrivateNoStore())
		router.GET("/me", func(c *gin.Context) { c.AbortWithStatus(status) })
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/me", nil))
		require.Equal(t, status, recorder.Code)
		require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
		require.ElementsMatch(t, []string{"Origin", "Authorization"}, recorder.Header().Values("Vary"))
	}
}
