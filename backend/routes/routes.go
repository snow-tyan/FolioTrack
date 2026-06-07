package routes

import (
	"foliotrack/controllers"
	"foliotrack/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware intercepts requests and validates the JWT Token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, 40101, "未提供 Authorization 请求头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			utils.Error(c, http.StatusUnauthorized, 40102, "Authorization 格式不正确，应为 Bearer <Token>")
			c.Abort()
			return
		}

		claims, err := utils.ValidateJWT(parts[1])
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, 40103, "Token 无效或已过期")
			c.Abort()
			return
		}

		// Save claims to context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// CORSMiddleware allows cross-origin requests in development
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Apply CORS
	r.Use(CORSMiddleware())

	api := r.Group("/api")
	{
		// Public Auth
		auth := api.Group("/auth")
		{
			auth.POST("/register", controllers.Register)
			auth.POST("/login", controllers.Login)
		}

		// Protected Routes
		protected := api.Group("")
		protected.Use(AuthMiddleware())
		{
			// Auth Profile
			protected.GET("/auth/me", controllers.Me)

			// Holdings
			protected.GET("/holdings", controllers.ListHoldings)
			protected.POST("/holdings", controllers.CreateHolding)
			protected.PUT("/holdings/:id", controllers.UpdateHolding)
			protected.DELETE("/holdings/:id", controllers.DeleteHolding)

			// Import / Export
			protected.GET("/holdings/export", controllers.ExportHoldings)
			protected.POST("/holdings/import", controllers.ImportHoldings)

			// Combos
			protected.GET("/combos", controllers.ListCombos)
			protected.POST("/combos", controllers.CreateCombo)
			protected.PUT("/combos/:id", controllers.UpdateCombo)
			protected.DELETE("/combos/:id", controllers.DeleteCombo)

			// Indexes
			protected.GET("/market/indexes", controllers.GetMarketIndexes)
		}
	}

	return r
}
