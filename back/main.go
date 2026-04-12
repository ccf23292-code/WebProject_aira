package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/routers"
	"warehouse-web/services"
)

func main() {
	// ── 初始化数据库与回忆卷服务 ──────────────────
	db, err := services.InitPostgres()
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.AuthSession{},
		&models.EmailVerification{},
		&models.Course{},
		&models.Teacher{},
		&models.TeacherSubmission{},
		&models.GradingStandard{},
		&models.GradingStandardSubmission{},
		&models.TeacherComment{},
		&models.CourseComment{},
		&models.CourseDescriptionSubmission{},
		&models.HomepageMessage{},
		&models.TestPaper{},
		&models.Problem{},
		&models.Favorite{},
		&models.AnswerRecord{},
		&models.WrongQuestion{},
		&models.ProblemExplanation{},
		&models.ProblemExplanationVote{},
	); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}
	// ── 初始化服务层 ──────────────────────────────
	var mailer services.Mailer
	if !isVerificationEchoEnabled() {
		smtpConfig, err := services.LoadSMTPConfigFromEnv()
		if err != nil {
			log.Fatalf("smtp init failed: %v", err)
		}
		mailer = services.NewSMTPMailer(smtpConfig)
	}

	authService := services.NewAuthService(db, mailer)
	paperService := services.NewPaperService(db)
	courseService := services.NewCourseService(db)
	homepageService := services.NewHomepageService(db)
	recallService := services.NewRecallService(db)
	favoriteService := services.NewFavoriteService(db, paperService)
	answerService := services.NewAnswerService(db, paperService)
	wrongBookService := services.NewWrongBookService(db, paperService)
	profileService := services.NewProfileService(db)
	explanationService := services.NewProblemExplanationService(db, paperService)
	if err := recallService.AutoMigrate(); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	// ── 初始化控制器 ──────────────────────────────
	authCtl := routers.NewAuthController(authService)
	paperCtl := routers.NewPaperController(paperService, courseService)
	courseCtl := routers.NewCourseController(courseService)
	homepageCtl := routers.NewHomepageController(homepageService)
	favoriteCtl := routers.NewFavoriteController(favoriteService)
	adminCtl := routers.NewAdminController(paperService, courseService)
	recallCtl := routers.NewRecallController(recallService)
	answerCtl := routers.NewAnswerController(answerService)
	wrongCtl := routers.NewWrongBookController(wrongBookService)
	profileCtl := routers.NewProfileController(profileService)
	explanationCtl := routers.NewProblemExplanationController(explanationService)

	// ── 创建 Gin 引擎 ─────────────────────────────
	r := gin.Default()
	r.Static("/static", "./storage")

	// 跨域配置（开发环境允许所有来源）
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// ── 路由注册 ──────────────────────────────────
	api := r.Group("/api")
	{
		// 1. auth_module —— 无需鉴权
		authGroup := api.Group("/auth")
		authCtl.RegisterRoutes(authGroup)

		// 2. browse_module —— 公开访问
		paperCtl.RegisterRoutes(api)
		homepageCtl.RegisterPublicRoutes(api)
		api.Use(middlewares.TryAuth(authService))
		explanationCtl.RegisterPublicRoutes(api)

		// 3. favorite_module —— 需要登录
		favGroup := api.Group("/favorites", middlewares.AuthRequired(authService))
		favoriteCtl.RegisterRoutes(favGroup)

		// 4. admin_module —— 需要登录 + 管理员权限
		adminGroup := api.Group("/admin",
			middlewares.AuthRequired(authService),
			middlewares.AdminRequired(),
		)
		adminCtl.RegisterRoutes(adminGroup)

		// 5. recall_module —— 回忆卷相关（需登录）
		recallGroup := api.Group("/recall", middlewares.AuthRequired(authService))
		recallCtl.RegisterRoutes(recallGroup)

		// 6. answer_module —— 做题记录（需登录）
		answerGroup := api.Group("/answers", middlewares.AuthRequired(authService))
		answerCtl.RegisterRoutes(answerGroup)

		// 7. wrongbook_module —— 错题本（需登录）
		wrongGroup := api.Group("/wrongbook", middlewares.AuthRequired(authService))
		wrongCtl.RegisterRoutes(wrongGroup)

		// 8. profile_module —— 用户资料（需登录）
		profileGroup := api.Group("/profile", middlewares.AuthRequired(authService))
		profileCtl.RegisterRoutes(profileGroup)

		// course_module -- course comments (auth required)
		courseGroup := api.Group("", middlewares.AuthRequired(authService))
		courseCtl.RegisterRoutes(courseGroup)

		homepageGroup := api.Group("", middlewares.AuthRequired(authService))
		homepageCtl.RegisterProtectedRoutes(homepageGroup)

		// 9. explanation_module —— 题解（公开读，登录后写/投票）
		explanationGroup := api.Group("", middlewares.AuthRequired(authService))
		explanationCtl.RegisterProtectedRoutes(explanationGroup)
	}

	// ── 启动服务 ──────────────────────────────────
	log.Println("AIRAWeb server starting on :3001 ...")
	if err := r.Run(":3001"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

func isVerificationEchoEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("DEV_EMAIL_ECHO")))
	return value == "1" || value == "true" || value == "yes"
}
