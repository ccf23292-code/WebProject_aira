package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"warehouse-web/services"
	"warehouse-web/utils"
)

// Context 键名，供下游 handler 读取已认证的用户信息。
const (
	CtxKeyUserID   = "userID"
	CtxKeyUsername = "username"
	CtxKeyRole     = "role"
)

// AuthRequired 返回一个鉴权中间件，校验 Authorization: Bearer <token>。
// 校验通过后将 userID、username、role 写入 gin.Context，
// 后续 handler 可通过 c.Get(middlewares.CtxKeyXxx) 获取。
func AuthRequired(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 预检请求由 CORS 中间件处理，这里直接放行避免 401。
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		if header == "" {
			utils.JSONError(c, http.StatusUnauthorized, "unauthorized", "缺少 Authorization 请求头")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authorization 格式应为 Bearer <token>")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			utils.JSONError(c, http.StatusUnauthorized, "unauthorized", "token 不能为空")
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			utils.JSONError(c, http.StatusUnauthorized, "unauthorized", "token 无效或已过期")
			return
		}

		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyUsername, claims.Username)
		c.Set(CtxKeyRole, string(claims.Role))
		c.Next()
	}
}

// TryAuth 在请求头存在有效 Bearer token 时写入用户上下文；否则直接放行。
// 该中间件用于“公开接口 + 登录态增强”的场景，例如返回 my_vote / my_item。
func TryAuth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			c.Next()
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err == nil {
			c.Set(CtxKeyUserID, claims.UserID)
			c.Set(CtxKeyUsername, claims.Username)
			c.Set(CtxKeyRole, string(claims.Role))
		}
		c.Next()
	}
}
