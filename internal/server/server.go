package server

import (
	"net/http"
	"path/filepath"
	"strconv"
	"totallyguysproject/internal/handlers"
	"totallyguysproject/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	Router *gin.Engine
}

func NewServer(db *gorm.DB) *Server {
	r := gin.Default()
	hub := ws.NewHub()
	// web static
	frontendPath := filepath.Join("..", "..", "..", "totallyweb", "web")
	r.Static("/static", frontendPath)
	r.Static("/uploads", filepath.Join("..", "..", "..", "totallyweb", "uploads"))

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		uidStr := c.Query("user_id")
		uid64, _ := strconv.ParseUint(uidStr, 10, 64)
		userID := uint(uid64)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		hub.AddClient(userID, conn)

		go func() {
			defer hub.RemoveClient(userID, conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	// API
	api := r.Group("/api")
	{
		// Movies
		movies := api.Group("/movies")
		movies.Use(handlers.AuthMiddleware(false))
		{
			movies.POST("/:id/like", func(c *gin.Context) { handlers.LikeMovie(c, db) })
			movies.DELETE("/:id/like", func(c *gin.Context) { handlers.UnlikeMovie(c, db) })

			// Reviews on moviepage

			movies.POST("/:id/reviews", func(c *gin.Context) { handlers.CreateReview(c, db, hub) })
		}
		api.GET("movies/load-by-genre", func(c *gin.Context) { handlers.LoadMoviesByGenre(c, db) })
		api.GET("movies/:id/reviews", func(c *gin.Context) { handlers.GetReviewsForMovie(c, db) })
		api.GET("/movies/search", func(c *gin.Context) { handlers.SearchAndSaveMovie(c, db) })
		api.GET("/movies/:id", func(c *gin.Context) { handlers.GetMovie(c, db) })

		reviews := api.Group("/reviews")
		reviews.Use(handlers.AuthMiddleware(false))
		{
			reviews.PUT("/:id", func(c *gin.Context) { handlers.UpdateReview(c, db) })
			reviews.DELETE("/:id", func(c *gin.Context) { handlers.DeleteReview(c, db) })

			//comments nested under reviews
			reviews.GET("/:id/comments", func(c *gin.Context) { handlers.GetCommentsForReview(c, db) })
			reviews.POST("/:id/comments", func(c *gin.Context) { handlers.CreateComment(c, db) })
		}
		// comments
		comments := api.Group("/comments")
		comments.Use(handlers.AuthMiddleware(false))
		{
			comments.PUT("/:id", func(c *gin.Context) { handlers.UpdateComment(c, db) })
			comments.DELETE("/:id", func(c *gin.Context) { handlers.DeleteComment(c, db) })
		}

		// Authentiication
		api.POST("/auth/register", func(c *gin.Context) { handlers.Register(c, db) })
		api.POST("/auth/login", func(c *gin.Context) { handlers.Login(c, db) })
		api.POST("/auth/logout", handlers.Logout)
		api.POST("/auth/verify", func(c *gin.Context) { handlers.VerifyEmail(c, db) })

		user := api.Group("/users")
		{
			user.GET("/:id/followers", func(c *gin.Context) { handlers.GetFollowersByID(c, db) })
			user.GET("/:id/following", func(c *gin.Context) { handlers.GetFollowingByID(c, db) })
			user.GET("/:id/reviews", func(c *gin.Context) { handlers.GetReviewsByUser(c, db) })
			userAuth := user.Group("/")
			userAuth.Use(handlers.AuthMiddleware(false))
			{
				// current user
				userAuth.GET("/me", func(c *gin.Context) { handlers.GetCurrentUser(c, db) })
				userAuth.DELETE("/me", func(c *gin.Context) { handlers.DeleteUser(c, db) })
				userAuth.PUT("/me", func(c *gin.Context) { handlers.UpdateCurrentUser(c, db) })
				userAuth.POST("/me/avatar", func(c *gin.Context) { handlers.UploadAvatar(c, db) })
				userAuth.DELETE("/me/avatar", func(c *gin.Context) { handlers.DeleteAvatar(c, db) })
				userAuth.GET("/me/playlists", func(c *gin.Context) { handlers.GetMyPlaylists(c, db) })
				userAuth.GET("/me/reviews", func(c *gin.Context) { handlers.GetMyReviews(c, db) })

				// follow/unfollow other users
				userAuth.GET("/me/followers", func(c *gin.Context) { handlers.GetMyFollowers(c, db) })
				userAuth.GET("/me/following", func(c *gin.Context) { handlers.GetMyFollowing(c, db) })
				userAuth.POST("/:id/follow", func(c *gin.Context) { handlers.FollowUser(c, db) })
				userAuth.DELETE("/:id/follow", func(c *gin.Context) { handlers.UnfollowUser(c, db) })

			}
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
			playlist.PUT("/:id/movies/:movie_id/description", func(c *gin.Context) {handlers.UpdateMovieDescriptionInPlaylist(c, db) })

			//playlist.POST("/:id/cover", func(c *gin.Context) { handlers.UploadPlaylistCover(c, db) }) // l8r

		}
		api.GET("playlists/:id", func(c *gin.Context) { handlers.GetPlaylist(c, db) })

	}

	// main page
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	return &Server{Router: r}
}
