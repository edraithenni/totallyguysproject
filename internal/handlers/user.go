package handlers

import (
	"fmt"
	"net/http"
	"os"
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
		var cover string
		switch p.Name {
		case "watched":
			cover = "/src/watched-playlist.jpg"
		case "watch-later":
			cover = "/src/watch-later-playlist.jpg"
		case "liked":
			cover = "/src/liked-playlist.jpg"
		default:
			cover = p.Cover
		}
		collections = append(collections, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": cover,
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
		//"friends":      friends,
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

	var user models.User
	if err := db.Preload("Playlists").Preload("Reviews").
		First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	playlists := []map[string]interface{}{}
	for _, p := range user.Playlists {
		var cover string
		switch p.Name {
		case "watched":
			cover = "/src/watched-playlist.jpg"
		case "watch-later":
			cover = "/src/watch-later-playlist.jpg"
		case "liked":
			cover = "/src/liked-playlist.jpg"
		default:
			cover = p.Cover
		}
		playlists = append(playlists, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"cover": cover,
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
		"email":       user.Email,
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
func deleteAvatarFile(avatarURL string) error {
	if avatarURL == "" {
		return nil
	}

	// ИСПРАВЛЕН ПУТЬ: вместо ../../../ используется абсолютный путь в контейнере
	filePath := "/app" + avatarURL

	fmt.Printf("Deleting: %s\n", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(filePath)
}

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

	// create an individual user directory
	// ИСПРАВЛЕН ПУТЬ: /app/uploads/avatars
	basePath := "/app/uploads/avatars"
	userFolder := fmt.Sprintf("%s/%v", basePath, userID)

	if err := os.MkdirAll(userFolder, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user folder"})
		return
	}

	// gen unique filename (userID + timestamp)
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	savePath := userFolder + "/" + filename

	// save to uploads
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// for frontend url
	avatarURL := fmt.Sprintf("/uploads/avatars/%v/%s", userID, filename)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	if user.Avatar != "" {
		go deleteAvatarFile(user.Avatar)
	}

	c.JSON(http.StatusOK, gin.H{"avatar": avatarURL})
}

func DeleteAvatar(c *gin.Context, db *gorm.DB) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	oldAvatar := user.Avatar

	if err := db.Model(&user).Update("avatar", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar"})
		return
	}

	if oldAvatar != "" {
		if err := deleteAvatarFile(oldAvatar); err != nil {
			fmt.Printf("Failed to delete avatar file: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"avatar": ""})
}

// @Summary Upload playlist cover
// @Description Uploads or replaces a cover image for a playlist owned by current user.
// @Tags playlists
// @Accept multipart/form-data
// @Produce json
// @Param cover formData file true "Cover file"
// @Param playlist_id path int true "Playlist ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me/playlists/{playlist_id}/cover [post]
func UploadPlaylistCover(c *gin.Context, db *gorm.DB) {
	userIDInterface, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDInterface.(uint)

	playlistIDParam := c.Param("playlist_id")
	if playlistIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playlist_id required"})
		return
	}

	pid, err := strconv.ParseUint(playlistIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}

	var playlist models.Playlist
	if err := db.First(&playlist, uint(pid)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}
	// Запретить изменение обложки для базовых плейлистов
	if playlist.Name == "watched" || playlist.Name == "watch-later" || playlist.Name == "liked" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нельзя менять обложку базового плейлиста"})
		return
	}

	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided"})
		return
	}

	// ИСПРАВЛЕН ПУТЬ: /app/uploads/playlists
	basePath := "/app/uploads/playlists"
	userFolder := fmt.Sprintf("%s/%v", basePath, userID)

	if err := os.MkdirAll(userFolder, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create folder"})
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	savePath := userFolder + "/" + filename

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	coverURL := fmt.Sprintf("/uploads/playlists/%v/%s", userID, filename)

	oldCover := playlist.Cover

	if err := db.Model(&models.Playlist{}).Where("id = ?", playlist.ID).Update("cover", coverURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update playlist cover"})
		return
	}

	if oldCover != "" {
		go func(url string) {
			// ИСПРАВЛЕН ПУТЬ: /app
			filePath := "/app" + url
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return
			}
			_ = os.Remove(filePath)
		}(oldCover)
	}

	c.JSON(http.StatusOK, gin.H{"cover": coverURL})
}

// @Summary Delete playlist cover
// @Description Removes cover image from playlist (sets to empty). Only owner can delete.
// @Tags playlists
// @Accept json
// @Produce json
// @Param playlist_id path int true "Playlist ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me/playlists/{playlist_id}/cover [delete]
func DeletePlaylistCover(c *gin.Context, db *gorm.DB) {
	userIDInterface, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDInterface.(uint)

	playlistIDParam := c.Param("playlist_id")
	if playlistIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playlist_id required"})
		return
	}

	pid, err := strconv.ParseUint(playlistIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}

	var playlist models.Playlist
	if err := db.First(&playlist, uint(pid)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}
	// Запретить удаление обложки для базовых плейлистов
	if playlist.Name == "watched" || playlist.Name == "watch-later" || playlist.Name == "liked" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нельзя менять обложку базового плейлиста"})
		return
	}

	oldCover := playlist.Cover

	if err := db.Model(&playlist).Update("cover", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update playlist"})
		return
	}

	if oldCover != "" {
		// ИСПРАВЛЕН ПУТЬ: /app
		filePath := "/app" + oldCover
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("Failed to delete playlist cover file: %v\n", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"cover": ""})
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
		{"watch-later", "/src/watch-later-playlist.jpg"},
		{"watched", "/src/watched-playlist.jpg"},
		{"liked", "/src/liked-playlist.jpg"},
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
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := db.First(&user, uid.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Delete related data: playlists, reviews, follows, comments
	// Use Unscoped to permanently delete if using soft deletes
	db.Unscoped().Where("owner_id = ?", user.ID).Delete(&models.Playlist{})
	db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.Review{})
	db.Unscoped().Where("follower_id = ? OR followed_id = ?", user.ID, user.ID).Delete(&models.Follow{})
	db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.Comment{})
	db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.CommentVote{})

	if err := db.Unscoped().Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	// Clear the authentication cookie
	c.SetCookie("token", "", -1, "/", "", false, true)

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

// handlers/user.go
func IsFollowing(c *gin.Context, db *gorm.DB) {
	currentUserInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	currentUser := currentUserInterface.(models.User)
	targetUserID := c.Param("id")

	targetID, err := strconv.ParseUint(targetUserID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var count int64
	db.Model(&models.Follow{}).Where("follower_id = ? AND followed_id = ?",
		currentUser.ID, uint(targetID)).Count(&count)

	c.JSON(http.StatusOK, gin.H{"isFollowing": count > 0})
}
