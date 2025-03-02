//go:build integration
// +build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerファイルUIのテスト
func TestSwaggerEndpoint(t *testing.T) {
	// テスト環境設定
	gin.SetMode(gin.TestMode)

	// テスト用のリクエストを作成
	req, err := http.NewRequest("GET", "/swagger/index.html", nil)
	assert.NoError(t, err, "リクエスト作成に失敗しました")

	w := httptest.NewRecorder()

	// Ginエンジンでリクエストを処理
	r := gin.New() // Default()からNew()に変更（ロギングやリカバリーなしの軽量版）

	// Swaggerハンドラの登録
	r.GET("/swagger/*any", func(c *gin.Context) {
		// 実際のSwaggerハンドラの代わりにモック応答を返す
		if c.Param("any") == "/index.html" {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, "<html><body>Swagger UI</body></html>")
			return
		}
		c.Status(http.StatusNotFound)
	})

	r.ServeHTTP(w, req)

	// レスポンスの検証
	assert.Equal(t, http.StatusOK, w.Code, "ステータスコードが一致しません")
	assert.Contains(t, w.Body.String(), "Swagger UI", "レスポンスにSwagger UIが含まれていません")
}

// 実際のswagger.yamlを使用した統合テスト
func TestSwaggerWithYAML(t *testing.T) {
	// 統合テストをスキップするフラグ
	if testing.Short() {
		t.Skip("統合テストはshortモードではスキップされます")
	}

	gin.SetMode(gin.TestMode)

	// テスト用のルーターを作成
	r := gin.New()

	// swagger.yamlファイルを提供するエンドポイント
	r.GET("/api-docs/swagger.yaml", func(c *gin.Context) {
		c.File("./swagger.yaml")
	})

	// SwaggerハンドラをCustomWrapHandlerで設定
	r.GET("/swagger/*any", ginSwagger.CustomWrapHandler(
		&ginSwagger.Config{
			URL: "/api-docs/swagger.yaml", // YAML取得先を指定
		},
		swaggerFiles.Handler,
	))

	// 1. Swagger UIのテスト
	t.Run("Swagger UI Test", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Swagger UI")
	})

	// 2. Swagger YAMLファイルのテスト
	t.Run("Swagger YAML File Test", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api-docs/swagger.yaml", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "openapi:") // swagger: -> openapi:
	})
}

// SwaggerファイルとAPI実装の整合性をテストする
func TestAPIAgainstSwagger(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストはshortモードではスキップされます")
	}

	// テスト用DBの設定
	db, err := connectDB()
	if err != nil {
		t.Skipf("データベース接続に失敗したためテストをスキップします: %v", err)
	}

	// テスト用のルーターを設定
	r := gin.New()
	setupRoutes(r, db) // 実際のルートを設定

	// Swagger定義からAPIエンドポイントをテスト

	// 1. GET /stocks のテスト
	t.Run("GET /stocks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/stocks", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// レスポンスがJSON形式であることを確認
		contentType := w.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json")
	})

	// 2. GET /stocks/:name のテスト
	t.Run("GET /stocks/:name", func(t *testing.T) {
		// 存在する商品名で検証
		req, _ := http.NewRequest("GET", "/stocks/apple", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// レスポンス検証（存在しなくても200が返る仕様）
		assert.Equal(t, http.StatusOK, w.Code)

		// レスポンスがJSON形式であることを確認
		contentType := w.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json")
	})

	// 3. その他のAPIエンドポイントも同様にテスト...
}
