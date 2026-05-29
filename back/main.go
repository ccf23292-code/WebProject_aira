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
		&models.UserCheckin{},
		&models.LLMExplanation{},
		&models.IngestJob{},
		&models.Message{},
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
	checkinService := services.NewCheckinService(db)
	messageService := services.NewMessageService(db)
	llmService := services.NewLLMService(services.LoadLLMConfigFromEnv(), db, paperService)
	if !llmService.Enabled() {
		log.Println("LLM service disabled: LLM_API_KEY not set, /api/llm/* will return 503")
	}
	visionClient := services.NewVisionClient(services.LoadVisionConfigFromEnv())
	if !visionClient.Enabled() {
		log.Println("Vision client disabled: LLM_VISION_API_KEY not set, image ingest will fail with vision_disabled")
	}
	ingestService := services.NewIngestService(db, llmService, visionClient, paperService)
	if err := recallService.AutoMigrate(); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	authCtl := routers.NewAuthController(authService)
	paperCtl := routers.NewPaperController(paperService, courseService)
	courseCtl := routers.NewCourseController(courseService)
	homepageCtl := routers.NewHomepageController(homepageService)
	favoriteCtl := routers.NewFavoriteController(favoriteService)
	adminCtl := routers.NewAdminController(paperService, courseService, recallService)
	recallCtl := routers.NewRecallController(recallService)
	answerCtl := routers.NewAnswerController(answerService)
	wrongCtl := routers.NewWrongBookController(wrongBookService)
	profileCtl := routers.NewProfileController(profileService)
	explanationCtl := routers.NewProblemExplanationController(explanationService)
	checkinCtl := routers.NewCheckinController(checkinService)
	messageCtl := routers.NewMessageController(messageService)
	llmCtl := routers.NewLLMController(llmService)
	fileCtl := routers.NewFileController()
	ingestCtl := routers.NewIngestController(ingestService)

	if err := os.MkdirAll("storage", 0o755); err != nil {
		log.Fatalf("create storage dir failed: %v", err)
	}

	r := gin.Default()
	r.MaxMultipartMemory = 20 << 20
	r.Static("/static", "./storage")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))
	r.Use(middlewares.RequireHTTPS())

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		authCtl.RegisterRoutes(authGroup)

		paperCtl.RegisterRoutes(api)
		homepageCtl.RegisterPublicRoutes(api)
		api.Use(middlewares.TryAuth(authService))
		explanationCtl.RegisterPublicRoutes(api)

		favGroup := api.Group("/favorites", middlewares.AuthRequired(authService))
		favoriteCtl.RegisterRoutes(favGroup)

		adminGroup := api.Group(
			"/admin",
			middlewares.AuthRequired(authService),
			middlewares.AdminRequired(),
		)
		adminCtl.RegisterRoutes(adminGroup)

		recallGroup := api.Group("/recall", middlewares.AuthRequired(authService))
		recallCtl.RegisterRoutes(recallGroup)

		answerGroup := api.Group("/answers", middlewares.AuthRequired(authService))
		answerCtl.RegisterRoutes(answerGroup)

		wrongGroup := api.Group("/wrongbook", middlewares.AuthRequired(authService))
		wrongCtl.RegisterRoutes(wrongGroup)

		profileGroup := api.Group("/profile", middlewares.AuthRequired(authService))
		profileCtl.RegisterRoutes(profileGroup)

		checkinGroup := api.Group("/checkin", middlewares.AuthRequired(authService))
		checkinCtl.RegisterRoutes(checkinGroup)

		messageGroup := api.Group("/messages", middlewares.AuthRequired(authService))
		messageCtl.RegisterRoutes(messageGroup)

		llmGroup := api.Group("/llm", middlewares.AuthRequired(authService))
		llmCtl.RegisterRoutes(llmGroup)

		fileGroup := api.Group("/files", middlewares.AuthRequired(authService))
		fileCtl.RegisterRoutes(fileGroup)

		// 上传清洗模块：普通用户挂 /api/ingest，admin 挂 /api/admin/ingest
		ingestUserGroup := api.Group("/ingest", middlewares.AuthRequired(authService))
		ingestCtl.RegisterUserRoutes(ingestUserGroup)
		ingestCtl.RegisterAdminRoutes(adminGroup)

		courseGroup := api.Group("", middlewares.AuthRequired(authService))
		courseCtl.RegisterRoutes(courseGroup)

		homepageGroup := api.Group("", middlewares.AuthRequired(authService))
		homepageCtl.RegisterProtectedRoutes(homepageGroup)

		// 9. explanation_module —— 题解（公开读，登录后写/投票）
		explanationGroup := api.Group("", middlewares.AuthRequired(authService))
		explanationCtl.RegisterProtectedRoutes(explanationGroup)
	}

	log.Println("AIRAWeb server starting on :3001 ...")
	if err := r.Run(":3001"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

func isVerificationEchoEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("DEV_EMAIL_ECHO")))
	return value == "1" || value == "true" || value == "yes"
}
