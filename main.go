package main

import (
	"fmt"
	_ "lambda-api-gw-go/docs" // swaggoが生成したSwaggerドキュメントをインポート
	"log"
	"os"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	r := gin.Default()
	setupRoutes(r, db) // ルーティング設定を routes.go に移動

	// ドキュメントを提供するエンドポイント
	r.GET("/api-docs/swagger.yaml", func(c *gin.Context) {
		c.File("./swagger.yaml")
	})

	// 環境変数からホスト名とポートを取得するか、デフォルト値を使用
	host := os.Getenv("APP_HOST")
	// fmt.Println("host::::--------------", db) //debug
	fmt.Println(host)
	if host == "" {
		host = "192.168.1.78" // デフォルト値
	}

	port := os.Getenv("APP_PORT")
	// fmt.Println("port::::--------------", port) //debug
	if port == "" {
		port = "8080" // デフォルト値
	}

	serverURL := fmt.Sprintf("http://%s:%s", host, port)

	// Swagger UIのエンドポイントを追加
	// http://192.168.1.78:8080/api-docs/swagger.yaml にアクセスすると、Swagger UI が表示される
	r.GET("/swagger/*any", ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL: serverURL + "/api-docs/swagger.yaml",
	}, swaggerFiles.Handler))

	// サーバーを起動し、エラーをチェック
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
