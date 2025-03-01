package main

import (
	"fmt"
	_ "lambda-api-gw-go/docs" // swaggoが生成したSwaggerドキュメントをインポート
	"log"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	fmt.Print(db) //debug
	r := gin.Default()
	setupRoutes(r, db) // ルーティング設定を routes.go に移動

	// ドキュメントを提供するエンドポイント
	r.GET("/api-docs/swagger.yaml", func(c *gin.Context) {
		c.File("./swagger.yaml")
	})

	// Swagger UIのエンドポイントを追加
	r.GET("/swagger/*any", ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL: "http://192.168.1.78:8080/api-docs/swagger.yaml",
	}, swaggerFiles.Handler))

	// Swagger UIのエンドポイントを追加
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}
