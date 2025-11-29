package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func MakeAuthMiddleware(authParams *AuthParams) gin.HandlerFunc {
	if authParams == nil {
		authParams = &AuthParams{
			Username: "admin",
			Password: "admin",
		}
	}
	return func(ctx *gin.Context) {
		if authParams.Username != ctx.GetHeader("Username") || authParams.Password != ctx.GetHeader("Password") {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
