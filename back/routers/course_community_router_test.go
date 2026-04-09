package routers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"warehouse-web/models"
	"warehouse-web/services"
)

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func newCourseRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:course-router-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserProfile{}, &models.Course{}, &models.Teacher{}, &models.CourseComment{}, &models.TeacherComment{}, &models.GradingStandard{}, &models.TestPaper{}, &models.Problem{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newCourseRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	paperCtl := NewPaperController(services.NewPaperService(db), services.NewCourseService(db))
	api := r.Group("/api")
	paperCtl.RegisterRoutes(api)
	return r
}

func TestListCoursesAcceptsQueryAlias(t *testing.T) {
	db := newCourseRouterTestDB(t)
	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/courses?query=CS1018F", nil)
	rec := httptest.NewRecorder()
	newCourseRouter(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiEnvelope[[]models.Course]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected one filtered course, got %#v", resp.Data)
	}
}

func TestCourseCommunityReadRoutesAreRegistered(t *testing.T) {
	db := newCourseRouterTestDB(t)
	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := db.Create(&models.Teacher{ID: "t-li", CourseID: "CS1018F", Name: "李老师"}).Error; err != nil {
		t.Fatalf("create teacher: %v", err)
	}

	engine := newCourseRouter(t, db)
	endpoints := []string{
		"/api/courses/CS1018F/comments",
		"/api/courses/CS1018F/teachers",
		"/api/courses/CS1018F/teachers/t-li/comments",
		"/api/courses/CS1018F/teachers/t-li/grading-standards",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected route %s to be registered, got 404", endpoint)
		}
	}
}
