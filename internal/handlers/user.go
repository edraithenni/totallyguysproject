package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"totallyguysproject/internal/models"
	"totallyguysproject/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ultranasral
type UpdateUserRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PlaylistsResponse struct {
	Playlists []PlaylistSummary `json:"playlists"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Avatar      string `json:"avatar"`
	Description string `json:"description"`
	Role        string `json:"role"`
}

type CurrentUserResponse struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Email       string            `json:"email"`
	Avatar      string            `json:"avatar"`
	Description string            `json:"description"`
	Role        string            `json:"role"`
	Collections []PlaylistSummary `json:"collections"`
	Followers   []uint            `json:"followers"`
	Following   []uint            `json:"following"`
	Reviews     []ReviewSummary   `json:"reviews"`
}

type PlaylistSummary struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Cover string `json:"cover"`
}

type ReviewSummary struct {
	ID      uint   `json:"id"`
	MovieID uint   `json:"movieId"`
	Content string `json:"content"`
	Rating  int    `json:"rating"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type AvatarResponse struct {
	Avatar string `json:"avatar"`
}

// @Summary Get current user
// @Description GET api/users/me.
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} handlers.CurrentUserResponse
// @Failure 401 {object} map[string]string
// @Router /users/me [get]
func GetCurrentUser(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var user models.User
	if err := db.Preload("Playlists").Preload("Reviews").First(&user, uid.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// user playlists for frontend
	collections := []map[string]interface{}{}
	for _, p := range user.Playlists {
		collections = append(collections, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": "/static/collection-placeholder.png",
		})
	}
	// user friends for frontend
	//friends := []uint{}
	//for _, f := range user.Friends {
	//  friends = append(friends, f.ID)
	//}
	var following []models.Follow
	db.Where("follower_id = ?", user.ID).Find(&following)

	var followingIDs []uint
	for _, f := range following {
		followingIDs = append(followingIDs, f.FollowedID)
	}

	var followers []models.Follow
	db.Where("followed_id = ?", user.ID).Find(&followers)

	var followerIDs []uint
	for _, f := range followers {
		followerIDs = append(followerIDs, f.FollowerID)
	}

	// user reviews for frontend
	reviews := []map[string]interface{}{}
	for _, r := range user.Reviews {
		reviews = append(reviews, map[string]interface{}{
			"id":      r.ID,
			"movieId": r.MovieID,
			"content": r.Content,
			"rating":  r.Rating,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"role":        user.Role, //nado li ?
		"email":       user.Email,
		"avatar":      user.Avatar,
		"description": user.Description,
		"collections": collections,
		//"friends":     friends,
		"following": followingIDs,
		"followers": followerIDs,
		"reviews":   reviews,
	})
}

type SearchUsersResponse struct {
	Users []UserResponse `json:"users"`
}

// @Summary Update current user
// @Description PUT api/users/me.
// @Tags users
// @Accept json
// @Produce json
// @Param body body UpdateUserRequest true "User data"
// @Success 200 {object} handlers.MessageResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/me [put]
func UpdateCurrentUser(c *gin.Context, db *gorm.DB) {
	uid, _ := c.Get("userID")
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user models.User
	if err := db.First(&user, uid.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Description != "" {
		user.Description = req.Description
	}

	db.Save(&user)
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// @Summary Get user profile
// @Description GET api/users/:id
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /users/{id} [get]
func GetProfile(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	uid, exists := c.Get("userID")
	if exists && id == fmt.Sprintf("%v", uid) {
		// redirecting to /me
		GetCurrentUser(c, db)
		return
	}

	var user models.User
	if err := db.Preload("Playlists").Preload("Reviews").
		First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	playlists := []map[string]interface{}{}
	for _, p := range user.Playlists {
		playlists = append(playlists, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": p.Cover,
		})
	}

	reviews := []map[string]interface{}{}
	for _, r := range user.Reviews {
		reviews = append(reviews, map[string]interface{}{
			"id":      r.ID,
			"movieId": r.MovieID,
			"content": r.Content,
			"rating":  r.Rating,
		})
	}

	//  friends := []uint{}
	// for _, f := range user.Friends {
	//   friends = append(friends, f.ID)
	//}
	var following []models.Follow
	db.Where("follower_id = ?", user.ID).Find(&following)

	var followingIDs []uint
	for _, f := range following {
		followingIDs = append(followingIDs, f.FollowedID)
	}

	var followers []models.Follow
	db.Where("followed_id = ?", user.ID).Find(&followers)

	var followerIDs []uint
	for _, f := range followers {
		followerIDs = append(followerIDs, f.FollowerID)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"avatar":      user.Avatar,
		"description": user.Description,
		"playlists":   playlists,
		"following":   followingIDs,
		"followers":   followerIDs,
		"reviews":     reviews,
	})
}

// @Summary Search users
// @Description GET api/users/search
// @Tags users
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Success 200 {object} handlers.SearchUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/search [get]
func SearchUsers(c *gin.Context, db *gorm.DB) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}

	var users []models.User
	if err := db.Where("name ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%").
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search users"})
		return
	}

	safeUsers := []gin.H{}
	for _, u := range users {
		safeUsers = append(safeUsers, gin.H{
			"id":     u.ID,
			"name":   u.Name,
			"email":  u.Email,
			"avatar": u.Avatar,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": safeUsers})
}

// not ready yet
// @Summary Upload avatar
// @Description loads cur user avatar.
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar file"
// @Success 200 {object} handlers.AvatarResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me/avatar [post]
func UploadAvatar(c *gin.Context, db *gorm.DB) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided"})
		return
	}

	// gen unique filename (userID + timestamp)
	filename := fmt.Sprintf("user_%v_%d_%s", userID, time.Now().Unix(), file.Filename)
	savePath := "./uploads/" + filename

	// save to uploads
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// for frontend url
	avatarURL := "/uploads/" + filename

	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"avatar": avatarURL})
}

// @Summary Create user (admin)
// @Description Admin create user(alternative for register if not done yet).
// @Tags users
// @Accept json
// @Produce json
// @Param body body CreateUserRequest true "New user"
// @Success 201 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [post]
func CreateUser(c *gin.Context, db *gorm.DB) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// exists check
	var exists int64
	db.Model(&models.User{}).Where("email = ?", req.Email).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user already exists"})
		return
	}

	hashed, _ := utils.HashPassword(req.Password)

	user := models.User{
		Name:        req.Name,
		Email:       req.Email,
		Password:    hashed,
		Role:        req.Role,
		Verified:    false,
		Avatar:      "",
		Description: "",
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	defaultPlaylists := []struct {
		Name  string
		Cover string
	}{
		{"watch-later", "/static/playlists/watch-later.png"},
		{"watched", "/static/playlists/watched.png"},
		{"liked", "/static/playlists/liked.png"},
	}

	for _, p := range defaultPlaylists {
		db.Create(&models.Playlist{
			Name:    p.Name,
			OwnerID: user.ID,
			Cover:   p.Cover,
		})
	}

	c.JSON(http.StatusCreated, user)
}

func DeleteUser(c *gin.Context, db *gorm.DB) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user models.User
	if err := db.First(&user, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	//delete user related data
	db.Where("owner_id = ?", user.ID).Delete(&models.Playlist{})
	db.Where("user_id = ?", user.ID).Delete(&models.Review{})
	db.Where("follower_id = ? OR followed_id = ?", user.ID, user.ID).Delete(&models.Follow{})
	db.Where("user_id = ?", user.ID).Delete(&models.Comment{})

	if err := db.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	// Clear auth cookie if the deleted user is the current user
	c.SetCookie("token", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// @Summary Get my playlists
// @Description GET api/users/me/playlists
// @Tags playlists
// @Accept json
// @Produce json
// @Success 200 {object} handlers.PlaylistsResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me/playlists [get]
func GetMyPlaylists(c *gin.Context, db *gorm.DB) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var playlists []models.Playlist
	if err := db.Where("owner_id = ?", uid.(uint)).Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch playlists"})
		return
	}

	resp := []map[string]interface{}{}
	for _, p := range playlists {
		resp = append(resp, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": p.Cover,
		})
	}

	c.JSON(http.StatusOK, gin.H{"playlists": resp})
}

// @Summary Get user playlists
// @Description  GET api/users/:id/playlists
// @Tags playlists
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} handlers.PlaylistsResponse
// @Failure 500 {object} map[string]string
// @Router /users/{id}/playlists [get]
func GetUserPlaylists(c *gin.Context, db *gorm.DB) {
	userID := c.Param("id")

	var playlists []models.Playlist
	if err := db.Where("owner_id = ?", userID).Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch playlists"})
		return
	}

	resp := []map[string]interface{}{}
	for _, p := range playlists {
		resp = append(resp, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": p.Cover,
		})
	}

	c.JSON(http.StatusOK, gin.H{"playlists": resp})
}
