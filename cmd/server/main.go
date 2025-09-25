// @title TotallyGuysProject API
// @version 1.0
// @description API для TotallyGuysProject
// @host localhost:8080
// @BasePath /api
package main

import (
    "totallyguysproject/internal/server"
    //"totallyguysproject/internal/models"
	"totallyguysproject/internal/database"
	docs "totallyguysproject/docs"
    swaggerfiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	db := database.InitDB()
    
    srv := server.NewServer(db)
	//docs.SwaggerInfo.BasePath = "/api/v1"
    //docs.SwaggerInfo.BasePath = ""
	// @BasePath /api
	docs.SwaggerInfo.BasePath = "/api"
    
	srv.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler,
    ginSwagger.URL("http://localhost:8080/swagger/doc.json"),
    ginSwagger.DefaultModelsExpandDepth(-1)))
    srv.Router.Run(":8080")
}