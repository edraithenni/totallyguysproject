package server

import (
	"path/filepath"
	"totallyguysproject/internal/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

)

type Server struct {
	Router *gin.Engine
}

func NewServer(db *gorm.DB) *Server {
	r := gin.Default()


	// web static
	frontendPath := filepath.Join("..", "..", "web")
	r.Static("/static", frontendPath)
	r.Static("/uploads", "./uploads")

	// API
	api := r.Group("/api")
	{
		// Movies
        api.GET("/movies/search", func(c *gin.Context) { handlers.SearchAndSaveMovie(c, db) })
        api.GET("/movies/:id", func(c *gin.Context) { handlers.GetMovie(c, db) })
		// Authentiication
		api.POST("/auth/register", func(c *gin.Context) { handlers.Register(c, db) })
		api.POST("/auth/login", func(c *gin.Context) { handlers.Login(c, db) })
		api.POST("/auth/logout", handlers.Logout)
		api.POST("/auth/verify", func(c *gin.Context) { handlers.VerifyEmail(c, db) })
		
		// cur user jwt	
		user := api.Group("/users")
		user.Use(handlers.AuthMiddleware(false))
		{
			user.GET("/me", func(c *gin.Context) { handlers.GetCurrentUser(c, db) })
			user.PUT("/me", func(c *gin.Context) { handlers.UpdateCurrentUser(c, db) })
			user.POST("/me/avatar", func(c *gin.Context) { handlers.UploadAvatar(c, db) })
		}
		//other users search (no jwt)
		api.GET("/users/search", func(c *gin.Context) { handlers.SearchUsers(c, db) })
        api.GET("/users/:id", func(c *gin.Context) { handlers.GetProfile(c, db) })

	}

	// main page 
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	return &Server{Router: r}
}
