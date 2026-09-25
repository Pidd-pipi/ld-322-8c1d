package middleware

import (
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/handler"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func Auth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
			handler.Fail(c, apperrors.New(apperrors.ErrUnauthorized.Code, "未登录或缺少令牌，无法保存校准偏移", apperrors.ErrUnauthorized.Status))
			c.Abort()
			return
		}
		claims, err := auth.Parse(parts[1])
		if err != nil {
			handler.Fail(c, apperrors.New(apperrors.ErrUnauthorized.Code, "登录已失效，请重新登录后再保存校准偏移", apperrors.ErrUnauthorized.Status))
			c.Abort()
			return
		}
		c.Set(constants.ContextClaims, claims)
		c.Next()
	}
}
