package handlers_test

import (

	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"totallyguysproject/internal/handlers"
)

func TestGetCurrentUser_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "avatar", "description", "role"}).
			AddRow(1, "Test User", "test@example.com", "/avatar.jpg", "Test description", "user"))

	mock.ExpectQuery(`SELECT \* FROM "playlists"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "cover", "owner_id"}).
			AddRow(1, "watch-later", "/cover.jpg", 1))

	mock.ExpectQuery(`SELECT \* FROM "reviews"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "movie_id", "content", "rating", "user_id"}).
			AddRow(1, 1, "Great movie", 5, 1))

	mock.ExpectQuery(`SELECT \* FROM "follows"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"follower_id", "followed_id"}).AddRow(1, 2))

	mock.ExpectQuery(`SELECT \* FROM "follows"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"follower_id", "followed_id"}).AddRow(3, 1))

	c, w := createTestContext("GET", "")
	c.Set("userID", uint(1))

	handlers.GetCurrentUser(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCurrentUser_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(1, "Old Name", "Old description"))

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	jsonData := `{"name":"New Name","description":"New description"}`
	c, w := createTestContext("PUT", jsonData)
	c.Set("userID", uint(1))

	handlers.UpdateCurrentUser(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchUsers_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs("%test%", "%test%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "avatar"}).
			AddRow(1, "Test User", "test@example.com", "/avatar.jpg").
			AddRow(2, "Test User2", "test2@example.com", "/avatar2.jpg"))

	c, w := createTestContext("GET", "")
	c.Request = httptest.NewRequest("GET", "/?query=test", nil)

	handlers.SearchUsers(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfile_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs("1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "avatar", "description"}).
			AddRow(1, "Test User", "test@example.com", "/avatar.jpg", "Test description"))

	mock.ExpectQuery(`SELECT \* FROM "playlists"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "cover", "owner_id"}).
			AddRow(1, "watch-later", "/cover.jpg", 1))

	mock.ExpectQuery(`SELECT \* FROM "reviews"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "movie_id", "content", "rating", "user_id"}).
			AddRow(1, 1, "Great movie", 5, 1))

	mock.ExpectQuery(`SELECT \* FROM "follows"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"follower_id", "followed_id"}).AddRow(1, 2))

	mock.ExpectQuery(`SELECT \* FROM "follows"`).WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"follower_id", "followed_id"}).AddRow(3, 1))

	c, w := createTestContext("GET", "")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handlers.GetProfile(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}