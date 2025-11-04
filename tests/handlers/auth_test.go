package handlers_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"totallyguysproject/internal/handlers"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	return gormDB, mock
}

func createTestContext(method, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	if body != "" {
		c.Request = httptest.NewRequest(method, "/", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, "/", nil)
	}
	
	return c, w
}

func TestRegister_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT count`).WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	for i := 0; i < 3; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "playlists"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i+1))
		mock.ExpectCommit()
	}

	jsonData := `{"name":"Test User","email":"test@example.com","password":"password123"}`
	c, w := createTestContext("POST", jsonData)

	handlers.Register(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_InvalidEmail(t *testing.T) {
	db, _ := setupTestDB(t)

	jsonData := `{"name":"Test User","email":"invalid-email","password":"password123"}`
	c, w := createTestContext("POST", jsonData)

	handlers.Register(c, db)

	assert.Equal(t, 400, w.Code)
}

func TestRegister_ShortPassword(t *testing.T) {
	db, _ := setupTestDB(t)

	jsonData := `{"name":"Test User","email":"test@example.com","password":"123"}`
	c, w := createTestContext("POST", jsonData)

	handlers.Register(c, db)

	assert.Equal(t, 400, w.Code)
}

func TestLogin_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	realHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs("test@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password", "verified"}).
			AddRow(1, "Test User", "test@example.com", string(realHash), true))

	jsonData := `{"email":"test@example.com","password":"password123"}`
	c, w := createTestContext("POST", jsonData)

	handlers.Login(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_UserNotFound(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs("none@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	jsonData := `{"email":"none@example.com","password":"password123"}`
	c, w := createTestContext("POST", jsonData)

	handlers.Login(c, db)

	assert.Equal(t, 401, w.Code)
}

func TestVerifyEmail_Success(t *testing.T) {
	db, mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs("test@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "verification_code", "verified"}).
			AddRow(1, "test@example.com", "123456", false))

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	jsonData := `{"email":"test@example.com","code":"123456"}`
	c, w := createTestContext("POST", jsonData)

	handlers.VerifyEmail(c, db)

	assert.Equal(t, 200, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}