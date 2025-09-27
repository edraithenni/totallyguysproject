package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"totallyguysproject/internal/models"
	"strconv"
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
        req.Cover = "/static/playlists/collection-placeholder.png"
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

    c.JSON(http.StatusCreated, playlist)
}

// GET /api/playlists/:id
func GetPlaylist(c *gin.Context, db *gorm.DB) {
    id := c.Param("id")

    var playlist models.Playlist
    if err := db.Preload("Movies").First(&playlist, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
        return
    }

    c.JSON(http.StatusOK, playlist)
}

// POST /api/playlists/:id/add
func AddMovieToPlaylist(c *gin.Context, db *gorm.DB) {
	//authenticated only
	uidRaw, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidRaw.(uint)

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
	uidRaw, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidRaw.(uint)

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
    uidRaw, ok := c.Get("userID")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    userID := uidRaw.(uint)

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
	uidRaw, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidRaw.(uint)

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
	uidRaw, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidRaw.(uint)

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


