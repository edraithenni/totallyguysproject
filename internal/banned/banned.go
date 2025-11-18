package banned

import (
	"sync"
	"totallyguysproject/internal/models"
	"gorm.io/gorm"
)

var (
	bannedUsers = make(map[uint]bool)
	mu          sync.RWMutex
	db          *gorm.DB
)

func Init(dbConn *gorm.DB) {
	db = dbConn
	var bans []models.BannedUser
	db.Find(&bans)
	mu.Lock()
	for _, b := range bans {
		bannedUsers[b.UserID] = true
	}
	mu.Unlock()
}

func BanUser(userID uint) error {
	mu.Lock()
	bannedUsers[userID] = true
	mu.Unlock()

	return db.Create(&models.BannedUser{UserID: userID}).Error
}

func UnbanUser(userID uint) error {
	mu.Lock()
	delete(bannedUsers, userID)
	mu.Unlock()

	return db.Delete(&models.BannedUser{}, "user_id = ?", userID).Error
}

func IsBanned(userID uint) bool {
	mu.RLock()
	defer mu.RUnlock()
	return bannedUsers[userID]
}
