package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name             string     `json:"name"`
	Email            string     `json:"email" gorm:"uniqueIndex"`
	Password         string     `json:"password"`
	Role             string     `json:"role"` // guest(no token)/user/admin (l8r)
	Verified         bool       `json:"verified"`
	VerificationCode string     `json:"verification_code"`
	Avatar           string     `json:"avatar"`
	Description      string     `json:"description"`
	Playlists        []Playlist `gorm:"foreignKey:OwnerID"` // FK
	Friends          []*User    `gorm:"many2many:user_friends;joinForeignKey:UserID;joinReferences:FriendID"`
	Reviews          []Review   `gorm:"foreignKey:UserID"`
}
type Movie struct {
	gorm.Model
    OMDBID   string  `json:"omdb_id" gorm:"uniqueIndex"` // imdb id
	//TMDbID   uint   `json:"tmdb_id"`
    Title    string  `json:"title"`
    Year     string  `json:"year"`
    Plot     string  `json:"plot"`
    Poster   string  `json:"poster"`
    Genre    string  `json:"genre"`
    Director string  `json:"director"`
    Actors   string  `json:"actors"`
    Rating   string  `json:"rating"`    // imdbRating string
    AvgRating float64 `json:"avg_rating"`
}

type Playlist struct {
	gorm.Model
	Name    string  `json:"name"`
	Cover   string  `json:"cover"`
	OwnerID uint    `json:"owner_id"`  //FK
	Movies  []Movie `gorm:"many2many:playlist_movies"`
}

type Review struct {
    gorm.Model
    MovieID uint   `json:"movie_id" gorm:"uniqueIndex:idx_user_movie"`
    UserID  uint   `json:"user_id" gorm:"uniqueIndex:idx_user_movie"`
    Content string `json:"content"`
    Rating  int    `json:"rating"` // 1-10
    Comments []Comment `gorm:"foreignKey:ReviewID"` 
}
//
type Comment struct {
    gorm.Model
    ReviewID uint   `json:"review_id"` 
    UserID   uint   `json:"user_id"`
    Content  string `json:"content"`
	//forum features
	ParentID *uint     `json:"parent_id"` //nullable fk
    Replies  []Comment `gorm:"foreignKey:ParentID" json:"replies"`
	//upvotes downvotes 
	Value     int   `json:"value"`
}
