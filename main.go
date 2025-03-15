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

// アプリケーション設定を一元管理する構造体
type AppConfig struct {
	Host string
	Port string
	// その他の設定...
}

// 環境変数から設定を読み込む
func loadConfig() AppConfig {
	config := AppConfig{
		Host: os.Getenv("APP_HOST"),
		Port: os.Getenv("APP_PORT"),
	}

	// デフォルト値の設定
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == "" {
		config.Port = "8080"
	}

	return config
}

func main() {
	// 設定を読み込む
	config := loadConfig()

	// ルーター設定（設定を渡す）
	r := setupRouter(config)

	// サーバー起動
	log.Fatal(r.Run(":" + config.Port))
}

func setupRouter(config AppConfig) *gin.Engine {
	r := gin.Default()

	// DB接続
	db, err := connectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// ルート設定
	setupRoutes(r, db)

	// ドキュメントを提供するエンドポイント
	r.GET("/api-docs/swagger.yaml", func(c *gin.Context) {
		c.File("./swagger.yaml")
	})

	// 設定値を使用
	serverURL := fmt.Sprintf("http://%s:%s", config.Host, config.Port)

	// Swagger UIのエンドポイントを追加
	// http://192.168.1.78:8080/api-docs/swagger.yaml にアクセスすると、Swagger UI が表示される
	r.GET("/swagger/*any", ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL: serverURL + "/api-docs/swagger.yaml",
	}, swaggerFiles.Handler))

	return r
}
