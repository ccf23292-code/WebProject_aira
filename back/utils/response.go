package utils

import "github.com/gin-gonic/gin"

// APIResponse 是所有接口的统一响应格式。
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// JSONSuccess 写入标准成功响应。
func JSONSuccess(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{
		Code:    status,
		Message: "success",
		Data:    data,
	})
}

// JSONSuccessMsg 写入带自定义消息的成功响应。
func JSONSuccessMsg(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, APIResponse{
		Code:    status,
		Message: message,
		Data:    data,
	})
}

// JSONError 写入标准错误响应并终止后续处理。
func JSONError(c *gin.Context, status int, code, message string, details ...interface{}) {
	c.AbortWithStatusJSON(status, APIResponse{
		Code:    status,
		Message: message,
		Data:    nil,
	})
}
