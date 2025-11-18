package handlers

import (
	"fmt"
	"net/http"
	"totallyguysproject/internal/models"
	"totallyguysproject/internal/utils"

	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// POST /api/auth/register
func Register(c *gin.Context, db *gorm.DB) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Trim spaces
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// validate email
	if _, err := mail.ParseAddress(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}

	// validate password
	if utf8.RuneCountInString(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	// validate name
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
		return
	}

	// exists - > error
	var exists int64
	db.Model(&models.User{}).Where("email = ?", req.Email).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user already exists"})
		return
	}

	hashed, _ := utils.HashPassword(req.Password)
	code := utils.GenerateVerificationCode(6)

	user := models.User{
		Name:             req.Name,
		Email:            req.Email,
		Password:         hashed,
		Role:             "user",
		Verified:         false,
		VerificationCode: code,
		Avatar:           "",
		Description:      "",
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// default playlists
	defaultPlaylists := []struct {
		Name  string
		Cover string
	}{
		{"watch-later", "/static/playlists/watch-later.png"},
		{"watched", "/static/playlists/watched.png"},
		{"liked", "/static/playlists/liked.png"},
	}

	for _, p := range defaultPlaylists {
		db.Create(&models.Playlist{
			Name:    p.Name,
			OwnerID: user.ID,
			Cover:   p.Cover,
		})
	}
	// verification through fmt, l8r with email
	fmt.Printf("Verification code for %s: %s\n", user.Email, code)

	c.JSON(http.StatusOK, gin.H{
		"message": "registered successfully, check console for verification code",
		"email":   user.Email,
	})
}

// POST /api/auth/login
func Login(c *gin.Context, db *gorm.DB) {

	if tokenCookie, err := c.Cookie("token"); err == nil && tokenCookie != "" {
		_, err := utils.ParseJWT(tokenCookie)
		if err == nil {
			// token is valid -> prohibit login
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "already logged in",
			})
			return
		}
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
		return
	}

	if !user.Verified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email not verified"})
		return
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", false, true)
//	c.SetCookie("token", token, 3600*24, "/", "", true, true)
//	c.Writer.Header().Add("Set-Cookie", fmt.Sprintf(
//    "token=%s; Path=/; Domain=.; Max-Age=%d; HttpOnly; SameSite=None",
//    token, 3600*24,
//))

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token":   token,
		//return token if client doesnt support cookies
	})
}

// POST /api/auth/logout
func Logout(c *gin.Context) {
	//logged out -> prohibit logout
	tokenCookie, err := c.Cookie("token")
	if err != nil || tokenCookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "not logged in",
		})
		return
	}
	//delete token
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// create admin (not used)
func EnsureAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hashed, _ := utils.HashPassword("admin123")
		admin := models.User{
			Name:        "Admin",
			Email:       "admin@site.com",
			Password:    hashed,
			Role:        "admin",
			Verified:    true,
			Avatar:      "",
			Description: "Admin user",
		}
		db.Create(&admin)

		defaultPlaylists := []string{"watch-later", "watched", "liked"}
		for _, name := range defaultPlaylists {
			db.Create(&models.Playlist{
				Name:    name,
				OwnerID: admin.ID,
			})
		}

		fmt.Println("Admin user created: admin@site.com / admin123")
	}
}

// POST /api/auth/verify
func VerifyEmail(c *gin.Context, db *gorm.DB) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.VerificationCode != req.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification code"})
		return
	}

	user.Verified = true
	user.VerificationCode = ""
	db.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}


//password reset funcs 
// POST /api/auth/forgot-password
func ForgotPassword(c *gin.Context, db *gorm.DB) {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	code := utils.GenerateVerificationCode(6)
	user.VerificationCode = code
	db.Save(&user)

	fmt.Printf("Password reset code for %s: %s\n", user.Email, code)

	c.JSON(http.StatusOK, gin.H{
		"message": "verification code sent (check console for verification code)",
	})
}

// POST /api/auth/reset-password
func ResetPassword(c *gin.Context, db *gorm.DB) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if utf8.RuneCountInString(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code or email"})
		return
	}

	if user.VerificationCode == "" || user.VerificationCode != req.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired code"})
		return
	}

	hashed, _ := utils.HashPassword(req.NewPassword)
	user.Password = hashed
	user.VerificationCode = ""
	db.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}