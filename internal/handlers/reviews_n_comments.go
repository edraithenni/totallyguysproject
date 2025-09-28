package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"totallyguysproject/internal/models"
)

// POST /api/movies/:id/reviews
func CreateReview(c *gin.Context, db *gorm.DB) {
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

    var reviews []models.Review
    if err := db.Where("movie_id = ?", movieID).Find(&reviews).Error; err != nil {
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

    var reviews []models.Review
    if err := db.Where("user_id = ?", userID).Find(&reviews).Error; err != nil {
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

    var reviews []models.Review
    if err := db.Where("user_id = ?", userID).Find(&reviews).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load my reviews"})
        return
    }

    c.JSON(http.StatusOK, reviews)
}


