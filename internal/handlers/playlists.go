package handlers

import (
	"net/http"
	"strconv"
	"totallyguysproject/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// POST /api/playlists authorized only
func CreatePlaylist(c *gin.Context, db *gorm.DB) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Name  string `json:"name"`
		Cover string `json:"cover"` //
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Cover == "" {
		req.Cover = "/src/default-playlist.jpg"
	}

	playlist := models.Playlist{
		Name:    req.Name,
		OwnerID: userID.(uint),
		Cover:   req.Cover,
	}
	if err := db.Create(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create playlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    playlist.ID,
		"name":  playlist.Name,
		"cover": playlist.Cover,
	})

}

// GET /api/playlists/:id
func GetPlaylist(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var playlist models.Playlist
	if err := db.First(&playlist, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	var owner models.User
	if err := db.First(&owner, playlist.OwnerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load owner"})
		return
	}

	type MovieWithDescription struct {
		models.Movie
		Description string `json:"description"`
	}

	var movies []MovieWithDescription
	if err := db.Table("movies").
		Select("movies.*, playlist_movies.description").
		Joins("JOIN playlist_movies ON playlist_movies.movie_id = movies.id").
		Where("playlist_movies.playlist_id = ?", playlist.ID).
		Scan(&movies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load movies"})
		return
	}

	//static covers for default playlists
	var cover string
	switch playlist.Name {
	case "watched":
		cover = "/src/watched-playlist.jpg"
	case "watch-later":
		cover = "/src/watch-later-playlist.jpg"
	case "liked":
		cover = "/src/liked-playlist.jpg"
	default:
		cover = playlist.Cover
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         playlist.ID,
		"name":       playlist.Name,
		"ownerId":    playlist.OwnerID,
		"movies":     movies,
		"owner_name": owner.Name,
		"cover":      cover,
	})
}

// POST /api/playlists/:id/add
func AddMovieToPlaylist(c *gin.Context, db *gorm.DB) {
	//authenticated only
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	// parse playlist id from path
	playlistIDStr := c.Param("id")
	pid64, err := strconv.ParseUint(playlistIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}
	playlistID := uint(pid64)

	// body: accept either "movie_id" or "omdb_id"
	var req struct {
		MovieID uint   `json:"movie_id"`
		OMDBID  string `json:"omdb_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// load playlist with movies
	var playlist models.Playlist
	if err := db.Preload("Movies").First(&playlist, playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	// ownership check
	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your playlist"})
		return
	}

	// find movie
	var movie models.Movie
	if req.MovieID != 0 {
		if err := db.First(&movie, req.MovieID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
	} else if req.OMDBID != "" {
		if err := db.Where("omdb_id = ?", req.OMDBID).First(&movie).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found by omdb id"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "movie_id or omdb_id required"})
		return
	}

	// check if already in playlist
	for _, m := range playlist.Movies {
		if m.ID == movie.ID {
			c.JSON(http.StatusOK, gin.H{"message": "movie already in playlist", "movie": movie})
			return
		}
	}

	// append association
	if err := db.Model(&playlist).Association("Movies").Append(&movie); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add movie to playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "movie added", "movie": movie})
}

// DELETE /api/playlists/:id
func DeletePlaylist(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	playlistIDStr := c.Param("id")
	pid64, err := strconv.ParseUint(playlistIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}
	playlistID := uint(pid64)

	var playlist models.Playlist
	if err := db.First(&playlist, playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your playlist"})
		return
	}

	if err := db.Model(&playlist).Association("Movies").Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear playlist movies"})
		return
	}

	if err := db.Unscoped().Delete(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "playlist deleted"})
}

// DELETE /api/playlists/:id/movies/:movie_id
func RemoveMovieFromPlaylist(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	playlistIDStr := c.Param("id")
	pid64, err := strconv.ParseUint(playlistIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}
	playlistID := uint(pid64)

	movieIDStr := c.Param("movie_id")
	mid64, err := strconv.ParseUint(movieIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	movieID := uint(mid64)

	var playlist models.Playlist
	if err := db.Preload("Movies").First(&playlist, playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your playlist"})
		return
	}

	var movie models.Movie
	if err := db.First(&movie, movieID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if err := db.Model(&playlist).Association("Movies").Unscoped().Delete(&movie); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "movie removed from playlist"})
}

// POST /api/movies/:id/like
func LikeMovie(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	movieID := c.Param("id")

	var movie models.Movie
	if err := db.First(&movie, movieID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	// find default playlist "liked"
	var playlist models.Playlist
	if err := db.Where("owner_id = ? AND name = ?", userID, "liked").Preload("Movies").First(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find liked playlist"})
		return
	}

	for _, m := range playlist.Movies {
		if m.ID == movie.ID {
			c.JSON(http.StatusOK, gin.H{"message": "already liked", "movie": movie})
			return
		}
	}

	if err := db.Model(&playlist).Association("Movies").Append(&movie); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "liked", "movie": movie})
}

// DELETE /api/movies/:id/like
func UnlikeMovie(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	movieID := c.Param("id")

	var movie models.Movie
	if err := db.First(&movie, movieID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	var playlist models.Playlist
	if err := db.Where("owner_id = ? AND name = ?", userID, "liked").Preload("Movies").First(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find liked playlist"})
		return
	}

	if err := db.Model(&playlist).Association("Movies").Unscoped().Delete(&movie); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unliked", "movie": movie})
}

// PUT /api/playlists/:id/movies/:movie_id/description
func UpdateMovieDescriptionInPlaylist(c *gin.Context, db *gorm.DB) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	pidStr := c.Param("id")
	midStr := c.Param("movie_id")

	pid64, err := strconv.ParseUint(pidStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist id"})
		return
	}
	mid64, err := strconv.ParseUint(midStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	playlistID := uint(pid64)
	movieID := uint(mid64)

	var playlist models.Playlist
	if err := db.First(&playlist, playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}
	if playlist.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your playlist"})
		return
	}

	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var link models.PlaylistMovie
	if err := db.Where("playlist_id = ? AND movie_id = ?", playlistID, movieID).First(&link).Error; err != nil {
		link = models.PlaylistMovie{
			PlaylistID:  playlistID,
			MovieID:     movieID,
			Description: req.Description,
		}
		if err := db.Create(&link).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create link"})
			return
		}
	} else {
		link.Description = req.Description
		if err := db.Save(&link).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update description"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "description updated",
		"description": link.Description,
	})
}
