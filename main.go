package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "lambda-api-gw-go/docs" // swaggoが生成したSwaggerドキュメントをインポート

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var ginLambda *ginadapter.GinLambda

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

func init() {
	// このinit関数はLambda起動時に一度だけ実行される
	log.Printf("Gin cold start")

	// 設定を読み込む
	config := loadConfig()

	// Lambdaでは本番モードが推奨
	gin.SetMode(gin.ReleaseMode)

	// ルーター設定
	r := setupRouter(config)

	// Ginアダプターを初期化
	ginLambda = ginadapter.New(r)
}

// Lambda用ハンドラー関数
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Ginルーターにリクエストを転送
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	// ローカル開発環境とLambda環境を判別
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		// Lambda環境ではLambdaハンドラを起動
		lambda.Start(Handler)
	} else {
		// ローカル環境では通常のHTTPサーバーを起動
		config := loadConfig()
		r := setupRouter(config)
		log.Printf("Starting local server on port %s", config.Port)
		log.Fatal(r.Run(":" + config.Port))
	}
}

func setupRouter(config AppConfig) *gin.Engine {
	r := gin.Default()

	// DB接続
	db, err := connectDB()
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		// Lambda環境ではエラーをログに出力するだけでプロセスを終了させない
		if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	}

	// ルート設定
	setupRoutes(r, db)

	// ドキュメントを提供するエンドポイント
	r.GET("/api-docs/swagger.yaml", func(c *gin.Context) {
		c.File("./swagger.yaml")
	})

	// APIゲートウェイのステージ名などを考慮したベースURL
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%s", config.Host, config.Port)
	}

	// Swagger UIのエンドポイントを追加
	r.GET("/swagger/*any", ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL: baseURL + "/api-docs/swagger.yaml",
	}, swaggerFiles.Handler))

	return r
}
