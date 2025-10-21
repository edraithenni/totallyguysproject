package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"totallyguysproject/internal/models"
	"encoding/csv"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var errorloadingenv = godotenv.Load()
var omdbAPIKey = os.Getenv("OMDB_API")

// GET /api/movies/search?title=...
func SearchAndSaveMovie(c *gin.Context, db *gorm.DB) {
	title := strings.TrimSpace(c.Query("title"))
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}

	exactMatch := c.Query("exact") == "true"
	//looking for exact match in db first
	var cached []models.Movie
	query := db.Model(&models.Movie{})

	if exactMatch {
		query = query.Where("LOWER(title) = LOWER(?)", title)
	} else {
		query = query.Where("title ILIKE ? OR title ILIKE ?",
			title+"%",
			"% "+title+"%")
	}

	if err := query.Find(&cached).Error; err == nil && len(cached) > 0 {
		//sorting by relevance
		sort.Slice(cached, func(i, j int) bool {
			iTitle := strings.ToLower(cached[i].Title)
			jTitle := strings.ToLower(cached[j].Title)

			//Priority: exact match > starts with > contains
			if iTitle == strings.ToLower(title) && jTitle != strings.ToLower(title) {
				return true
			}
			if strings.HasPrefix(iTitle, strings.ToLower(title)) &&
				!strings.HasPrefix(jTitle, strings.ToLower(title)) {
				return true
			}
			return cached[i].Title < cached[j].Title
		})

		results := make([]interface{}, len(cached))
		for i, m := range cached {
			results[i] = struct {
				ID        uint   `json:"id"`
				OMDBID    string `json:"omdb_id"`
				Title     string `json:"title"`
				Year      string `json:"year"`
				Poster    string `json:"poster"`
				Relevance string `json:"relevance,omitempty"`
			}{
				ID:     m.ID,
				OMDBID: m.OMDBID,
				Title:  m.Title,
				Year:   m.Year,
				Poster: m.Poster,
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"Search": results,
			"source": "database",
			"total":  len(results),
		})
		return
	}

	// not in db -> load from OMDb
	escapedTitle := url.QueryEscape(title)
	var omdbURL string
	if exactMatch {
		omdbURL = fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&s=%s&type=movie", omdbAPIKey, escapedTitle)
	} else {
		// regular search with pagination
		omdbURL = fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&s=%s&type=movie&page=1", omdbAPIKey, escapedTitle)
	}

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
		if errorMsg, ok := omdbResp["Error"].(string); ok && strings.Contains(errorMsg, "not found") {
			c.JSON(http.StatusOK, gin.H{
				"Search": []interface{}{},
				"source": "omdb",
				"total":  0,
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": omdbResp["Error"]})
		return
	}

	moviesData, ok := omdbResp["Search"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected OMDb response"})
		return
	}

	//sorting OMDb results by relevance
	sort.Slice(moviesData, func(i, j int) bool {
		iMap, iOk := moviesData[i].(map[string]interface{})
		jMap, jOk := moviesData[j].(map[string]interface{})
		if !iOk || !jOk {
			return false
		}

		iTitle := strings.ToLower(fmt.Sprintf("%v", iMap["Title"]))
		jTitle := strings.ToLower(fmt.Sprintf("%v", jMap["Title"]))
		searchTerm := strings.ToLower(title)

		//Priority: exact match > starts with > contains
		if iTitle == searchTerm && jTitle != searchTerm {
			return true
		}
		if strings.HasPrefix(iTitle, searchTerm) && !strings.HasPrefix(jTitle, searchTerm) {
			return true
		}
		return iTitle < jTitle
	})

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

	c.JSON(http.StatusOK, gin.H{
		"Search": results,
		"source": "omdb",
		"total":  len(results),
	})
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

	if movie.Plot == "" || movie.Genre == "" || movie.Director == "" {
		detailsURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&i=%s", omdbAPIKey, movie.OMDBID)
		resp, err := http.Get(detailsURL)
		if err == nil {
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			var details map[string]interface{}
			if err := json.Unmarshal(body, &details); err == nil && details["Response"] == "True" {

				updates := make(map[string]interface{})

				if movie.Plot == "" {
					movie.Plot = fmt.Sprintf("%v", details["Plot"])
					updates["plot"] = movie.Plot
				}
				if movie.Genre == "" {
					movie.Genre = fmt.Sprintf("%v", details["Genre"])
					updates["genre"] = movie.Genre
				}
				if movie.Director == "" {
					movie.Director = fmt.Sprintf("%v", details["Director"])
					updates["director"] = movie.Director
				}
				if movie.Actors == "" {
					movie.Actors = fmt.Sprintf("%v", details["Actors"])
					updates["actors"] = movie.Actors
				}
				if movie.Rating == "" {
					movie.Rating = fmt.Sprintf("%v", details["imdbRating"])
					updates["rating"] = movie.Rating
				}

				if len(updates) > 0 {
					db.Model(&movie).Updates(updates)
				}
			}
		}
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

// GET /api/movies/load-by-genre?genre=Action&page=1
func LoadMoviesByGenre(c *gin.Context, db *gorm.DB) {
    genre := c.Query("genre")
    if genre == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "genre required"})
        return
    }

    pageStr := c.Query("page")
    page, _ := strconv.Atoi(pageStr)
    if page < 1 {
        page = 1
    }

    limit := 5
    offset := (page - 1) * limit

    // CSV 
    filePath := fmt.Sprintf("../../data/%s.csv", strings.ToLower(genre))
    f, err := os.Open(filePath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "genre CSV not found"})
        return
    }
    defer f.Close()

    reader := csv.NewReader(f)
    rows, _ := reader.ReadAll()

    type csvMovie struct {
        Title string
        Year  string
    }
    var csvMovies []csvMovie
    for i, row := range rows {
        if i == 0 {
            continue 
        }
        if len(row) < 3 {
            continue
        }
        csvMovies = append(csvMovies, csvMovie{
            Title: row[1],
            Year:  row[2],
        })
    }

    start := offset
    end := offset + limit
    if start >= len(csvMovies) {
        c.JSON(http.StatusOK, []models.Movie{})
        return
    }
    if end > len(csvMovies) {
        end = len(csvMovies)
    }
    pageMovies := csvMovies[start:end]

    var movies []models.Movie
    for _, m := range pageMovies {
        var dbMovie models.Movie
        if err := db.Where("title = ? AND year = ?", m.Title, m.Year).First(&dbMovie).Error; err != nil {
            // fetch OMDb по title+year
            omdbURL := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&t=%s&y=%s", omdbAPIKey, url.QueryEscape(m.Title), m.Year)
            resp, err := http.Get(omdbURL)
            if err != nil {
                continue
            }
            body, _ := io.ReadAll(resp.Body)
            resp.Body.Close()

            var omdbResp map[string]interface{}
            if err := json.Unmarshal(body, &omdbResp); err != nil || omdbResp["Response"] != "True" {
                continue
            }

            dbMovie = models.Movie{
                Title:    m.Title,
                Year:     m.Year,
                OMDBID:   fmt.Sprintf("%v", omdbResp["imdbID"]),
                Poster:   fmt.Sprintf("%v", omdbResp["Poster"]),
                Plot:     fmt.Sprintf("%v", omdbResp["Plot"]),
                Genre:    fmt.Sprintf("%v", omdbResp["Genre"]),
                Director: fmt.Sprintf("%v", omdbResp["Director"]),
                Actors:   fmt.Sprintf("%v", omdbResp["Actors"]),
                Rating:   fmt.Sprintf("%v", omdbResp["imdbRating"]),
            }
            db.Create(&dbMovie)
        }
		
        movies = append(movies, dbMovie)
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

    c.JSON(http.StatusOK, results)
}

