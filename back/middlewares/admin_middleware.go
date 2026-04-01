package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"warehouse-web/models"
	"warehouse-web/utils"
)

// AdminRequired 返回一个管理员权限校验中间件。
// 必须放在 AuthRequired 之后使用，因为它依赖 Context 中已设置的 role 信息。
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 预检请求由 CORS 中间件处理，这里直接放行避免 403。
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		role, exists := c.Get(CtxKeyRole)
		if !exists {
			utils.JSONError(c, http.StatusUnauthorized, "unauthorized", "未通过身份认证")
			return
		}

		if models.Role(role.(string)) != models.RoleAdmin {
			utils.JSONError(c, http.StatusForbidden, "forbidden", "需要管理员权限")
			return
		}

		c.Next()
	}
}
