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
		movies := api.Group("/movies")
        movies.Use(handlers.AuthMiddleware(false)) 
        {
            movies.POST("/:id/like", func(c *gin.Context) { handlers.LikeMovie(c, db) })
            movies.DELETE("/:id/like", func(c *gin.Context) { handlers.UnlikeMovie(c, db) })
        }
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
			user.GET("/me/playlists", func(c *gin.Context) { handlers.GetMyPlaylists(c, db) }) 
		}
		api.GET("/users/:id/playlists", func(c *gin.Context) { handlers.GetUserPlaylists(c, db) }) 
		//other users search (no jwt)
		api.GET("/users/search", func(c *gin.Context) { handlers.SearchUsers(c, db) })
        api.GET("/users/:id", func(c *gin.Context) { handlers.GetProfile(c, db) })

		// playlists
		playlist := api.Group("/playlists")
        playlist.Use(handlers.AuthMiddleware(false))
        {
            playlist.POST("/", func(c *gin.Context) { handlers.CreatePlaylist(c, db) })
            playlist.POST("/:id/add", func(c *gin.Context) { handlers.AddMovieToPlaylist(c, db) })
			playlist.DELETE("/:id", func(c *gin.Context) { handlers.DeletePlaylist(c, db) })
			playlist.DELETE("/:id/movies/:movie_id", func(c *gin.Context) { handlers.RemoveMovieFromPlaylist(c, db) })

          //  playlist.POST("/:id/cover", func(c *gin.Context) { handlers.UploadPlaylistCover(c, db) }) // l8r
			
        }
		playlist.GET("/:id", func(c *gin.Context) { handlers.GetPlaylist(c, db) })

	}

	// main page 
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	return &Server{Router: r}
}
