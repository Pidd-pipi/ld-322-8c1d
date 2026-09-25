package middleware

import (
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RequireRole 校验 JWT 中的角色，角色不匹配时返回 403 并说明原因；需放在 Auth 之后使用。
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(constants.ContextKeyClaims)
		claims, ok := value.(jwt.MapClaims)
		if !exists || !ok || claims["role"] != role {
			handler.Fail(c, apperrors.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
