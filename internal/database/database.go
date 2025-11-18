package database

import (
    "fmt"
    "log"
    "os"
    "totallyguysproject/internal/models"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/joho/godotenv"
)

func InitDB() *gorm.DB {
    // from .env
    if err := godotenv.Load(); err != nil {
        fmt.Println(".env file not found, using system environment variables")
    }

    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        log.Fatal("DATABASE_URL is not set")
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("failed to connect to database:", err)
    }

    err = db.AutoMigrate(
        &models.User{},
        &models.Movie{},
        &models.Playlist{}, 
        &models.Review{}, 
        &models.Comment{}, 
        &models.Follow{}, 
        &models.Notification{}, 
        &models.CommentVote{},
        &models.BannedUser{},
    )
    if err != nil {
        log.Fatal("failed to migrate database:", err)
    }

	if !db.Migrator().HasColumn(&models.PlaylistMovie{}, "Description") {
        if err := db.Migrator().AddColumn(&models.PlaylistMovie{}, "Description"); err != nil {
            log.Fatal("failed to create column", err)
        }
    }

    fmt.Println("Database connected and migrated")
    return db
}