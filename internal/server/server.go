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
	//r.Static("/uploads", "./uploads")

	// API
	api := r.Group("/api")
	{
		// Movies
        api.GET("/movies/search", func(c *gin.Context) { handlers.SearchAndSaveMovie(c, db) })
        api.GET("/movies/:id", func(c *gin.Context) { handlers.GetMovie(c, db) })

	}

	// main page 
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	return &Server{Router: r}
}
