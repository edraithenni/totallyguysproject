package handlers

import (
    "encoding/json"
    "fmt"
    "io"
	"log"   
    "net/http"
    "net/url"
	"strconv" 
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
    /*var movie models.Movie
    if err := db.Where("title LIKE ?", "%"+title+"%").First(&movie).Error; err == nil {
        c.JSON(http.StatusOK, movie)
        return
    }*/
	var cached []models.Movie
    if err := db.Where("title LIKE ?", "%"+title+"%").Find(&cached).Error; err == nil && len(cached) > 0 {
        c.JSON(http.StatusOK, gin.H{"Search": cached})
        return
    }

    // not in db->load from omdb -> cache movie in db
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
            // Plot/Genre/Director/Actors/Rating пока пустые — потом подтянем в GetMovie
        }

        db.FirstOrCreate(&newMovie, models.Movie{OMDBID: newMovie.OMDBID})
        movies = append(movies, newMovie)
    }

    c.JSON(http.StatusOK, gin.H{"Search": movies})
}


// GET /api/movies/:id  //:id может быть numeric DB id или imdbID (tt...)

func GetMovie(c *gin.Context, db *gorm.DB) {
    id := c.Param("id")
    var movie models.Movie
    var found bool

    //try numeric id (primary key)
    if num, err := strconv.ParseUint(id, 10, 64); err == nil {
        if err := db.First(&movie, num).Error; err == nil {
            found = true
        }
    }

    // else find omdb_id (imdbID)
    if !found {
        if err := db.Where("omdb_id = ?", id).First(&movie).Error; err == nil {
            found = true
        }
    }

    // если не найден в БД — пробуем сразу подтянуть из OMDb и сохранить
    if !found {
        omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s&plot=short", omdbAPIKey, url.QueryEscape(id))
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

        movie = models.Movie{
            Title:    fmt.Sprintf("%v", omdbResp["Title"]),
            Year:     fmt.Sprintf("%v", omdbResp["Year"]),
            OMDBID:   fmt.Sprintf("%v", omdbResp["imdbID"]),
            Plot:     fmt.Sprintf("%v", omdbResp["Plot"]),
            Poster:   fmt.Sprintf("%v", omdbResp["Poster"]),
            Genre:    fmt.Sprintf("%v", omdbResp["Genre"]),
            Director: fmt.Sprintf("%v", omdbResp["Director"]),
            Actors:   fmt.Sprintf("%v", omdbResp["Actors"]),
            Rating:   fmt.Sprintf("%v", omdbResp["imdbRating"]),
        }

        if err := db.Create(&movie).Error; err != nil {
           log.Println("Ошибка при сохранении фильма в БД:", err)
        }
        c.JSON(http.StatusOK, movie)
        return
    }

    // если нашли в БД, а деталей нет — подтягиваем из OMDb и обновляем
    if movie.Plot == "" || movie.Plot == "N/A" {
        omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s&plot=short", omdbAPIKey, url.QueryEscape(movie.OMDBID))
        resp, err := http.Get(omdbURL)
        if err == nil {
            defer resp.Body.Close()
            body, _ := io.ReadAll(resp.Body)
            var omdbResp map[string]interface{}
            if err := json.Unmarshal(body, &omdbResp); err == nil && omdbResp["Response"] == "True" {
                movie.Plot = fmt.Sprintf("%v", omdbResp["Plot"])
                movie.Genre = fmt.Sprintf("%v", omdbResp["Genre"])
                movie.Director = fmt.Sprintf("%v", omdbResp["Director"])
                movie.Actors = fmt.Sprintf("%v", omdbResp["Actors"])
                movie.Rating = fmt.Sprintf("%v", omdbResp["imdbRating"])
                db.Save(&movie)
            }
        }
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