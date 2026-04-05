package main

import (
	"log"
	"os"

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
		&models.GradingStandard{},
		&models.TeacherComment{},
		&models.CourseComment{},
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

	authService := services.NewAuthService(db)
	paperService := services.NewPaperService(db)
	courseService := services.NewCourseService(db)
	recallService := services.NewRecallService(db)
	favoriteService := services.NewFavoriteService(db, paperService)
	answerService := services.NewAnswerService(db, paperService)
	wrongBookService := services.NewWrongBookService(db, paperService)
	profileService := services.NewProfileService(db)
	explanationService := services.NewProblemExplanationService(db, paperService)
	if err := recallService.AutoMigrate(); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	authCtl := routers.NewAuthController(authService)
	paperCtl := routers.NewPaperController(paperService, courseService)
	courseCtl := routers.NewCourseController(courseService)
	favoriteCtl := routers.NewFavoriteController(favoriteService)
	adminCtl := routers.NewAdminController(paperService)
	recallCtl := routers.NewRecallController(recallService)
	answerCtl := routers.NewAnswerController(answerService)
	wrongCtl := routers.NewWrongBookController(wrongBookService)
	profileCtl := routers.NewProfileController(profileService)
	explanationCtl := routers.NewProblemExplanationController(explanationService)
	fileCtl := routers.NewFileController()

	if err := os.MkdirAll("storage", 0o755); err != nil {
		log.Fatalf("create storage dir failed: %v", err)
	}

	r := gin.Default()
	r.MaxMultipartMemory = 20 << 20
	r.Static("/static", "./storage")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		authCtl.RegisterRoutes(authGroup)

		paperCtl.RegisterRoutes(api)
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

		fileGroup := api.Group("/files", middlewares.AuthRequired(authService))
		fileCtl.RegisterRoutes(fileGroup)

		courseGroup := api.Group("", middlewares.AuthRequired(authService))
		courseCtl.RegisterRoutes(courseGroup)

		explanationGroup := api.Group("", middlewares.AuthRequired(authService))
		explanationCtl.RegisterProtectedRoutes(explanationGroup)
	}

	log.Println("AIRAWeb server starting on :3001 ...")
	if err := r.Run(":3001"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
