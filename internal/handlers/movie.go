package handlers

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "totallyguysproject/internal/models"
    "github.com/joho/godotenv"
	"os"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)
//var omdbAPIKey = os.Getenv("OMDB_API_KEY")
//var omdbAPIKey = "a91fcb6"
var errorloadingenv = godotenv.Load()
var omdbAPIKey = os.Getenv("OMDB_API")

// GET /api/movies/search?title=...
func SearchAndSaveMovie(c *gin.Context, db *gorm.DB) {
    title := c.Query("title")
    if title == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
        return
    }

    // db cached
    var movie models.Movie
    if err := db.Where("title LIKE ?", "%"+title+"%").First(&movie).Error; err == nil {
        c.JSON(http.StatusOK, movie)
        return
    }

    // not in db->load from omdb -> cache movie in db
    escapedTitle := url.QueryEscape(title)
    omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s", omdbAPIKey, escapedTitle)

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

    newMovie := models.Movie{
        Title:  fmt.Sprintf("%v", omdbResp["Title"]),
        Year:   fmt.Sprintf("%v", omdbResp["Year"]),
        OMDBID: fmt.Sprintf("%v", omdbResp["imdbID"]),
        Plot:   fmt.Sprintf("%v", omdbResp["Plot"]),
        Poster: fmt.Sprintf("%v", omdbResp["Poster"]),
    }

    db.FirstOrCreate(&newMovie, models.Movie{OMDBID: newMovie.OMDBID})
    c.JSON(http.StatusOK, newMovie)
}

// GET /api/movies/:id
func GetMovie(c *gin.Context, db *gorm.DB) {
    id := c.Param("id")
    var movie models.Movie
    if err := db.First(&movie, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
        return
    }
    c.JSON(http.StatusOK, movie)
}

// not used yet 
func GetMovies(c *gin.Context, db *gorm.DB) {
	var movies []models.Movie
	db.Find(&movies)
	c.JSON(http.StatusOK, movies)
}

func CreateMovie(c *gin.Context, db *gorm.DB) {
	var movie models.Movie
	if err := c.BindJSON(&movie); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&movie)
	c.JSON(http.StatusCreated, movie)
}
