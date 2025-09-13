package main

import (
    "totallyguysproject/internal/server"
    //"totallyguysproject/internal/models"
	"totallyguysproject/internal/database"
  //  "gorm.io/driver/sqlite"
    //"gorm.io/gorm"
   // "log"
	//"fmt"
	//"gorm.io/driver/postgres"
)

func main() {
	db := database.InitDB()
    // db.Exec("DELETE FROM Users") // if we need to clear users table

    srv := server.NewServer(db)
    srv.Router.Run(":8080")
}
