package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/routers"
	"warehouse-web/services"
)

func main() {
	// ── 初始化服务层 ──────────────────────────────
	authService := services.NewAuthService()
	paperService := services.NewPaperService()

	// ── 初始化数据库与回忆卷服务 ──────────────────
	db, err := services.InitPostgres()
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	recallService := services.NewRecallService(db)
	if err := recallService.AutoMigrate(); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	// ── 初始化控制器 ──────────────────────────────
	authCtl := routers.NewAuthController(authService)
	paperCtl := routers.NewPaperController(paperService)
	favoriteCtl := routers.NewFavoriteController(paperService)
	adminCtl := routers.NewAdminController(paperService)
	recallCtl := routers.NewRecallController(recallService)

	// ── 创建 Gin 引擎 ─────────────────────────────
	r := gin.Default()

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
	}

	// ── 启动服务 ──────────────────────────────────
	log.Println("AIRAWeb server starting on :3001 ...")
	if err := r.Run(":3001"); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
