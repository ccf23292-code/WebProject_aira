package routers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
)

func newDescriptionRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:course-description-router-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Course{},
		&models.CourseDescriptionSubmission{},
		&models.User{},
		&models.UserProfile{},
		&models.TestPaper{},
		&models.Problem{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestCourseDescriptionRoutesAreRegistered(t *testing.T) {
	db := newDescriptionRouterTestDB(t)
	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	courseCtl := NewCourseController(services.NewCourseService(db))
	adminCtl := NewAdminController(services.NewPaperService(db), services.NewCourseService(db))

	api := r.Group("/api")
	withUser := api.Group("")
	withUser.Use(func(c *gin.Context) {
		c.Set(middlewares.CtxKeyUserID, models.PrimaryKey(8))
		c.Set(middlewares.CtxKeyRole, string(models.RoleStudent))
		c.Next()
	})
	courseCtl.RegisterRoutes(withUser)

	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		c.Set(middlewares.CtxKeyUserID, models.PrimaryKey(1))
		c.Set(middlewares.CtxKeyRole, string(models.RoleAdmin))
		c.Next()
	})
	adminCtl.RegisterRoutes(admin)

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/api/courses/CS1018F/description-submissions",
		bytes.NewBufferString(`{"content":"课程简介提案"}`),
	)
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	if postRec.Code == http.StatusNotFound {
		t.Fatalf("expected description submission route to be registered")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/courses/CS1018F/description-submissions/mine", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code == http.StatusNotFound {
		t.Fatalf("expected my submissions route to be registered")
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/course-description-submissions?status=pending", nil)
	adminRec := httptest.NewRecorder()
	r.ServeHTTP(adminRec, adminReq)
	if adminRec.Code == http.StatusNotFound {
		t.Fatalf("expected admin list route to be registered")
	}
}
