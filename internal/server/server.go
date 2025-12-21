package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"totallyguysproject/internal/handlers"
	"totallyguysproject/internal/utils"
	"totallyguysproject/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	//"github.com/joho/godotenv"
	"totallyguysproject/internal/logger"
)

func LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang, err := c.Cookie("lang")
		if err != nil {
			lang = c.GetHeader("Accept-Language")
			if lang == "" {
				lang = "en"
			}
		}
		if len(lang) > 2 {
			lang = lang[:2]
		}
		c.Set("language", lang)
		c.Next()
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowed := map[string]bool{
			"http://localhost:3000": true,
		}
		return allowed[origin]
	},
}

type Server struct {
	Router *gin.Engine
}

func NewServer(db *gorm.DB) *Server {

	logger.InitFileLogger()

	r := gin.Default()
	r.Use(logger.FileLoggerMiddleware())
	r.Use(LanguageMiddleware())
	hub := ws.NewHub(db)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept-Language"},
		AllowCredentials: true,
	}))
	ws.StartNotificationCleanup(db) //deletes checked notifications every hour
	// web static (legacy static content)
	r.Static("/legacy", "./public/legacy/web")
	r.Static("/uploads", "/app/uploads")

	// next js host
	nextURL, _ := url.Parse("http://localhost:3000")
	r.Any("/app/*path", func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(nextURL)
		c.Request.URL.Path = c.Param("path")
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// WebSocket endpoint
	// WebSocket endpoint — now authenticates tokens before upgrading.
	r.GET("/ws", func(c *gin.Context) {
		token := ""
		if cookie, err := c.Cookie("token"); err == nil && cookie != "" {
			token = cookie
		}
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					token = parts[1]
				}
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
			return
		}

		claims, err := utils.ParseJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		uidFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in token"})
			return
		}
		userID := uint(uidFloat)

		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			allowed := map[string]bool{
				"http://localhost:3000": true,
			}
			if !allowed[origin] {
				c.JSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
				return
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		hub.AddClient(userID, conn)

		hub.SendPendingFromDB(userID, conn)

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
		//admin endpoints
		admin := r.Group("/api/admin")
		admin.Use(handlers.AuthMiddleware(false), handlers.AdminRoleRequired())
		admin.GET("/users/:id/banned", func(c *gin.Context) {
			handlers.AdminGetUserBanStatus(c, db)
		})

		admin.DELETE("/reviews/:id", func(c *gin.Context) {
			handlers.AdminDeleteReview(c, db, hub)
		})
		admin.DELETE("/comments/:id", func(c *gin.Context) {
			handlers.AdminDeleteComment(c, db, hub)
		})
		admin.POST("/users/:id/ban", func(c *gin.Context) {
			handlers.AdminBanUser(c, hub)
		})

		admin.POST("/users/:id/unban", func(c *gin.Context) {
			handlers.AdminUnbanUser(c, hub)
		})

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
			reviews.GET("/:id", func(c *gin.Context) { handlers.GetReview(c, db) })

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
			comments.POST("/:id/vote", func(c *gin.Context) { handlers.VoteComment(c, db) })
		}

		// Authentiication
		api.POST("/auth/register", func(c *gin.Context) { handlers.Register(c, db) })
		api.POST("/auth/login", func(c *gin.Context) { handlers.Login(c, db) })
		api.POST("/auth/logout", handlers.Logout)
		api.POST("/auth/verify", func(c *gin.Context) { handlers.VerifyEmail(c, db) })
		// Password recovery
		api.POST("/auth/forgot-password", func(c *gin.Context) { handlers.ForgotPassword(c, db) })
		api.POST("/auth/reset-password", func(c *gin.Context) { handlers.ResetPassword(c, db) })

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
				// playlist covers for user's own playlists
				userAuth.POST("/me/playlists/:playlist_id/cover", func(c *gin.Context) { handlers.UploadPlaylistCover(c, db) })
				userAuth.DELETE("/me/playlists/:playlist_id/cover", func(c *gin.Context) { handlers.DeletePlaylistCover(c, db) })
				userAuth.GET("/me/playlists", func(c *gin.Context) { handlers.GetMyPlaylists(c, db) })
				userAuth.GET("/me/reviews", func(c *gin.Context) { handlers.GetMyReviews(c, db) })

				// follow/unfollow other users
				userAuth.GET("/me/followers", func(c *gin.Context) { handlers.GetMyFollowers(c, db) })
				userAuth.GET("/me/following", func(c *gin.Context) { handlers.GetMyFollowing(c, db) })
				userAuth.POST("/:id/follow", func(c *gin.Context) { handlers.FollowUser(c, db, hub) })
				userAuth.DELETE("/:id/follow", func(c *gin.Context) { handlers.UnfollowUser(c, db) })
				userAuth.GET("/:id/is-following", func(c *gin.Context) { handlers.CheckIsFollowing(c, db) })

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
			playlist.POST("", func(c *gin.Context) { handlers.CreatePlaylist(c, db) })
			playlist.POST("/:id/add", func(c *gin.Context) { handlers.AddMovieToPlaylist(c, db) })
			playlist.DELETE("/:id", func(c *gin.Context) { handlers.DeletePlaylist(c, db) })
			playlist.DELETE("/:id/movies/:movie_id", func(c *gin.Context) { handlers.RemoveMovieFromPlaylist(c, db) })
			playlist.PUT("/:id/movies/:movie_id/description", func(c *gin.Context) { handlers.UpdateMovieDescriptionInPlaylist(c, db) })

			//playlist.POST("/:id/cover", func(c *gin.Context) { handlers.UploadPlaylistCover(c, db) }) // l8r

		}
		api.GET("playlists/:id", func(c *gin.Context) { handlers.GetPlaylist(c, db) })

	}

	// main page
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/legacy/index.html")
	})

	return &Server{Router: r}
}
