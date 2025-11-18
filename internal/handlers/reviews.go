package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"totallyguysproject/internal/banned"
	"totallyguysproject/internal/models"
	"totallyguysproject/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewWithMovie struct {
	ID        uint      `json:"id"`
	MovieID   uint      `json:"movie_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `json:"content"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	MovieTitle      string `json:"movie_title"`
	ContainsSpoiler bool   `json:"contains_spoiler"`
}

type ReviewWithMovieAndUser struct {
	ID              uint      `json:"id"`
	MovieID         uint      `json:"movie_id"`
	MovieTitle      string    `json:"movie_title"`
	UserID          uint      `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserAvatar      string    `json:"user_avatar"`
	Content         string    `json:"content"`
	Rating          int       `json:"rating"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ContainsSpoiler bool      `json:"contains_spoiler"`
}

// POST /api/movies/:id/reviews
func CreateReview(c *gin.Context, db *gorm.DB, hub *ws.Hub) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	if banned.IsBanned(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are banned from posting"})
		return
	}

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
		Content         string `json:"content"`
		Rating          int    `json:"rating"`
		ContainsSpoiler bool   `json:"contains_spoiler"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Rating < 1 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	review := models.Review{
		MovieID:         movieID,
		UserID:          userID,
		Content:         req.Content,
		Rating:          req.Rating,
		ContainsSpoiler: req.ContainsSpoiler,
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
			"type":        "review",
			"author_id":   userID,
			"author_name": author.Name,
			"movie_id":    movieID,
			"movie_title": movie.Title,
			"text":        fmt.Sprintf("%s wrote a new review for a movie «%s»", author.Name, movie.Title),
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
			reviews.content, reviews.rating, reviews.created_at, reviews.updated_at, reviews.contains_spoiler
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

// GET /api/reviews/:id
func GetReview(c *gin.Context, db *gorm.DB) {

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

	var user models.User
	if err := db.First(&user, review.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var movie models.Movie
	if err := db.First(&movie, review.MovieID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	var commentsCount int64
	db.Model(&models.Comment{}).Where("review_id = ?", reviewID).Count(&commentsCount)

	c.JSON(http.StatusOK, gin.H{
		"id":               review.ID,
		"user_id":          review.UserID,
		"user_name":        user.Name,
		"user_avatar":      user.Avatar,
		"movie_id":         review.MovieID,
		"movie_title":      movie.Title,
		"content":          review.Content,
		"rating":           review.Rating,
		"contains_spoiler": review.ContainsSpoiler,
		"created_at":       review.CreatedAt,
		"comments_count":   commentsCount,
	})
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
		Content         string `json:"content"`
		Rating          int    `json:"rating"`
		ContainsSpoiler bool   `json:"contains_spoiler"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Rating < 1 || req.Rating > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	review.Content = req.Content
	review.Rating = req.Rating
	review.ContainsSpoiler = req.ContainsSpoiler

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

	if err := db.Exec(`
        DELETE FROM comment_votes 
        WHERE comment_id IN (SELECT id FROM comments WHERE review_id = ?)
    `, reviewID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment votes"})
		return
	}

	if err := db.Where("review_id = ?", reviewID).Unscoped().Delete(&models.Comment{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comments"})
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
        	reviews.content, reviews.rating, reviews.created_at, reviews.updated_at, reviews.contains_spoiler
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
        	reviews.content, reviews.rating, reviews.created_at, reviews.updated_at, reviews.contains_spoiler
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
