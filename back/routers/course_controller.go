package routers

import (
	"warehouse-web/services"
)

// CourseController 处理课程相关的浏览请求。
type CourseController struct {
	service *services.CourseService
}
