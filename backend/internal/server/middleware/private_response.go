package middleware

import "github.com/gin-gonic/gin"

// PrivateNoStore prevents browser and shared HTTP caches from storing personal
// responses. Install before authentication/guards so errors receive it as well.
func PrivateNoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-store")
		c.Writer.Header().Add("Vary", "Authorization")
		c.Next()
	}
}
