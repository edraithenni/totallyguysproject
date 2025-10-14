package handlers

import (
	"net/http"
	"strconv"
	"totallyguysproject/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"totallyguysproject/internal/ws"
	"fmt"
	"log"
)

// POST /users/:id/follow
func FollowUser(c *gin.Context, db *gorm.DB,  hub *ws.Hub) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	myID := uid.(uint)

	targetID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	targetID := uint(targetID64)

	if myID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot follow yourself"})
		return
	}

	var existing models.Follow
	err = db.Unscoped().Where("follower_id = ? AND followed_id = ?", myID, targetID).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			if err := db.Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore follow"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "followed"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "already following"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	follow := models.Follow{
		FollowerID: myID,
		FollowedID: targetID,
	}
	if err := db.Create(&follow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to follow"})
		return
	}

	var followerUser, targetUser models.User
	if err := db.First(&followerUser, myID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get follower info"})
		return
	}
	if err := db.First(&targetUser, targetID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get target user info"})
		return
	}

	msg := map[string]interface{}{
		"type":         "follow",
		"follower_id":  myID,
		"follower_name": followerUser.Name,
		"text":         fmt.Sprintf("%s started following you", followerUser.Name),
	}

	log.Println("Sending follow notification to:", targetID)

	hub.SendToMany([]uint{targetID}, msg)

	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

// DELETE /users/:id/follow
func UnfollowUser(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	myID := uid.(uint)

	targetID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	targetID := uint(targetID64)

	if err := db.Where("follower_id = ? AND followed_id = ?", myID, targetID).Delete(&models.Follow{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unfollow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

// GET /users/:id/is-following
func CheckIsFollowing(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	myID := uid.(uint)

	targetID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	targetID := uint(targetID64)

	var existing models.Follow
	err = db.Where("follower_id = ? AND followed_id = ?", myID, targetID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"isFollowing": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": true})
}

// GET /users/:id/followers
func GetFollowersByID(c *gin.Context, db *gorm.DB) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	userID := uint(id64)

	var followers []models.User
	db.Joins("JOIN follows ON follows.follower_id = users.id").
		Where("follows.followed_id = ? AND follows.deleted_at IS NULL", userID).
		Find(&followers)

	c.JSON(http.StatusOK, gin.H{"followers": followers})
}

// GET /users/:id/following
func GetFollowingByID(c *gin.Context, db *gorm.DB) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	userID := uint(id64)

	var following []models.User
	db.Joins("JOIN follows ON follows.followed_id = users.id").
		Where("follows.follower_id = ? AND follows.deleted_at IS NULL", userID).
		Find(&following)

	c.JSON(http.StatusOK, gin.H{"following": following})
}

// GET /users/me/followers
func GetMyFollowers(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	var followers []models.User
	db.Joins("JOIN follows ON follows.follower_id = users.id").
		Where("follows.followed_id = ? AND follows.deleted_at IS NULL", userID).
		Find(&followers)

	c.JSON(http.StatusOK, gin.H{"followers": followers})
}

// GET /users/me/following
func GetMyFollowing(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	var following []models.User
	db.Joins("JOIN follows ON follows.followed_id = users.id").
		Where("follows.follower_id = ? AND follows.deleted_at IS NULL", userID).
		Find(&following)

	c.JSON(http.StatusOK, gin.H{"following": following})
}
