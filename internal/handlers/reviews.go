package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"totallyguysproject/internal/models"
	"totallyguysproject/internal/ws"
	"fmt"
	"time"
	"log"
)

type ReviewWithMovie struct {
    ID        uint      `json:"id"`
    MovieID   uint      `json:"movie_id"`
    UserID    uint      `json:"user_id"`
    Content   string    `json:"content"`
    Rating    int       `json:"rating"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    MovieTitle string `json:"movie_title"`
}

type ReviewWithMovieAndUser struct {
    ID          uint      `json:"id"`
    MovieID     uint      `json:"movie_id"`
    MovieTitle  string    `json:"movie_title"`
    UserID      uint      `json:"user_id"`
    UserName    string    `json:"user_name"`
    UserAvatar  string    `json:"user_avatar"`
    Content     string    `json:"content"`
    Rating      int       `json:"rating"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}


// POST /api/movies/:id/reviews
func CreateReview(c *gin.Context, db *gorm.DB, hub *ws.Hub) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	movieIDStr := c.Param("id")
	movieID64, err := strconv.ParseUint(movieIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	movieID := uint(movieID64)

	var movie models.Movie
	if err := db.First(&movie, movieID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	var req struct {
		Content string `json:"content"`
		Rating  int    `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Rating < 1 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	review := models.Review{
		MovieID: movieID,
		UserID:  userID,
		Content: req.Content,
		Rating:  req.Rating,
	}

	if err := db.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}
    
	var followers []models.Follow
	if err := db.Where("followed_id = ?", userID).Find(&followers).Error; err == nil {
		followerIDs := make([]uint, 0, len(followers))
		for _, f := range followers {
			followerIDs = append(followerIDs, f.FollowerID)
		}

		var author models.User
		if err := db.First(&author, userID).Error; err != nil {
    		log.Println("no author info:", err)
		}

		msg := map[string]interface{}{
    		"type": "review",
    		"author_id": userID,
    		"author_name": author.Name,
    		"movie_id": movieID,
    		"movie_title": movie.Title,
    		"text": fmt.Sprintf("%s wrote a new review for a movie «%s»", author.Name, movie.Title),
		}

		hub.SendToMany(followerIDs, msg)

		//msg := fmt.Sprintf("User %d wrote review on film %d", userID, movieID)
		//hub.SendToMany(followerIDs, msg)
	}

	c.JSON(http.StatusCreated, review)
}

// GET /api/movies/:id/reviews
func GetReviewsForMovie(c *gin.Context, db *gorm.DB) {
    movieIDStr := c.Param("id")
    movieID64, err := strconv.ParseUint(movieIDStr, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
        return
    }
    movieID := uint(movieID64)

    var reviews []ReviewWithMovieAndUser
	if err := db.Table("reviews").
    	Select(`
        	reviews.id, reviews.movie_id, movies.title AS movie_title,
        	reviews.user_id, users.name AS user_name, users.avatar AS user_avatar,
        	reviews.content, reviews.rating, reviews.created_at, reviews.updated_at
    	`).
    	Joins("LEFT JOIN movies ON movies.id = reviews.movie_id").
    	Joins("LEFT JOIN users ON users.id = reviews.user_id").
    	Where("reviews.movie_id = ?", movieID).
    	Scan(&reviews).Error; err != nil {
    	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reviews"})
    	return
	}

	c.JSON(http.StatusOK, reviews)
}


// PUT /api/reviews/:id
func UpdateReview(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	reviewIDStr := c.Param("id")
	rid64, err := strconv.ParseUint(reviewIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	reviewID := uint(rid64)

	var review models.Review
	if err := db.First(&review, reviewID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}

	if review.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your review"})
		return
	}

	var req struct {
		Content string `json:"content"`
		Rating  int    `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Rating < 1 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	review.Content = req.Content
	review.Rating = req.Rating

	if err := db.Save(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review"})
		return
	}

	c.JSON(http.StatusOK, review)
}

// DELETE /api/reviews/:id
func DeleteReview(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	reviewIDStr := c.Param("id")
	rid64, err := strconv.ParseUint(reviewIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	reviewID := uint(rid64)

	var review models.Review
	if err := db.First(&review, reviewID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}

	if review.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your review"})
		return
	}

	if err := db.Unscoped().Delete(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "review deleted"})
}

// GET /api/users/:id/reviews
func GetReviewsByUser(c *gin.Context, db *gorm.DB) {
    userID := c.Param("id")

    var reviews []ReviewWithMovieAndUser
	if err := db.Table("reviews").
    	Select(`
        	reviews.id, reviews.movie_id, movies.title AS movie_title,
        	reviews.user_id, users.name AS user_name, users.avatar AS user_avatar,
        	reviews.content, reviews.rating, reviews.created_at, reviews.updated_at
    	`).
    	Joins("LEFT JOIN movies ON movies.id = reviews.movie_id").
    	Joins("LEFT JOIN users ON users.id = reviews.user_id").
    	Where("reviews.user_id = ?", userID).
    	Scan(&reviews).Error; err != nil {
    	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reviews"})
    	return
	}

    c.JSON(http.StatusOK, reviews)
}

// GET /api/users/me/reviews
func GetMyReviews(c *gin.Context, db *gorm.DB) {
    uid, ok := c.Get("userID")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    userID := uid.(uint)

    var reviews []ReviewWithMovieAndUser
	if err := db.Table("reviews").
    	Select(`
        	reviews.id, reviews.movie_id, movies.title AS movie_title,
        	reviews.user_id, users.name AS user_name, users.avatar AS user_avatar,
        	reviews.content, reviews.rating, reviews.created_at, reviews.updated_at
    	`).
    	Joins("LEFT JOIN movies ON movies.id = reviews.movie_id").
    	Joins("LEFT JOIN users ON users.id = reviews.user_id").
    	Where("reviews.user_id = ?", userID).
    	Scan(&reviews).Error; err != nil {
    	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reviews"})
    	return
	}

	c.JSON(http.StatusOK, reviews)
}


