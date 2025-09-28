package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"totallyguysproject/internal/models"
	"os"
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var errorloadingenv = godotenv.Load()
var omdbAPIKey = os.Getenv("OMDB_API")

// GET /api/movies/search?title=...
func SearchAndSaveMovie(c *gin.Context, db *gorm.DB) {
	title := c.Query("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}

	var cached []models.Movie
	if err := db.Where("title LIKE ?", "%"+title+"%").Find(&cached).Error; err == nil && len(cached) > 0 {
		results := make([]interface{}, len(cached))
		for i, m := range cached {
			results[i] = struct {
				ID     uint   `json:"id"`
				OMDBID string `json:"omdb_id"`
				Title  string `json:"title"`
				Year   string `json:"year"`
				Poster string `json:"poster"`
			}{
				ID:     m.ID,
				OMDBID: m.OMDBID,
				Title:  m.Title,
				Year:   m.Year,
				Poster: m.Poster,
			}
		}
		c.JSON(http.StatusOK, gin.H{"Search": results})
		return
	}

	// not in db -> load from OMDb
	escapedTitle := url.QueryEscape(title)
	omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&s=%s", omdbAPIKey, escapedTitle)

	resp, err := http.Get(omdbURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch OMDb"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var omdbResp map[string]interface{}
	if err := json.Unmarshal(body, &omdbResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse OMDb"})
		return
	}

	if omdbResp["Response"] != "True" {
		c.JSON(http.StatusNotFound, gin.H{"error": omdbResp["Error"]})
		return
	}

	moviesData, ok := omdbResp["Search"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected OMDb response"})
		return
	}

	var movies []models.Movie
	for _, m := range moviesData {
		mMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		newMovie := models.Movie{
			Title:  fmt.Sprintf("%v", mMap["Title"]),
			Year:   fmt.Sprintf("%v", mMap["Year"]),
			OMDBID: fmt.Sprintf("%v", mMap["imdbID"]),
			Poster: fmt.Sprintf("%v", mMap["Poster"]),
		}

		db.FirstOrCreate(&newMovie, models.Movie{OMDBID: newMovie.OMDBID})
		movies = append(movies, newMovie)
	}

	results := make([]interface{}, len(movies))
	for i, m := range movies {
		results[i] = struct {
			ID     uint   `json:"id"`
			OMDBID string `json:"omdb_id"`
			Title  string `json:"title"`
			Year   string `json:"year"`
			Poster string `json:"poster"`
		}{
			ID:     m.ID,
			OMDBID: m.OMDBID,
			Title:  m.Title,
			Year:   m.Year,
			Poster: m.Poster,
		}
	}

	c.JSON(http.StatusOK, gin.H{"Search": results})
}

// GET /api/movies/:id  db/local id
func GetMovie(c *gin.Context, db *gorm.DB) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}

	var movie models.Movie
	found := false

	if err := db.First(&movie, id).Error; err == nil {
		found = true
	}

	// not found in db -> try OMDb by OMDBID (optional)
	if !found {
		if err := db.Where("omdb_id = ?", idStr).First(&movie).Error; err == nil {
			found = true
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	c.JSON(http.StatusOK, struct {
		ID       uint   `json:"id"`
		OMDBID   string `json:"omdb_id"`
		Title    string `json:"title"`
		Year     string `json:"year"`
		Plot     string `json:"plot"`
		Poster   string `json:"poster"`
		Genre    string `json:"genre"`
		Director string `json:"director"`
		Actors   string `json:"actors"`
		Rating   string `json:"rating"`
	}{
		ID:       movie.ID,
		OMDBID:   movie.OMDBID,
		Title:    movie.Title,
		Year:     movie.Year,
		Plot:     movie.Plot,
		Poster:   movie.Poster,
		Genre:    movie.Genre,
		Director: movie.Director,
		Actors:   movie.Actors,
		Rating:   movie.Rating,
	})
}
