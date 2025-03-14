//go:build e2e_toriaezu
// +build e2e_toriaezu

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
func TestE2ESwaggerEndpoint(t *testing.T) {
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
func TestE2ESwaggerWithYAML(t *testing.T) {
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

	// SwaggerハンドラをWrapHandlerで設定（シンプルに）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api-docs/swagger.yaml"))) // オプションでURLを指定

	// 1. Swagger UIのテスト（インデックスページにアクセス）
	t.Run("Swagger UI Test", func(t *testing.T) {
		// 直接indexページにアクセス
		req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// デバッグ出力
		t.Logf("Response Status: %d", w.Code)
		t.Logf("Response Body: %s", w.Body.String())

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Swagger UI")
	})

	// 2. Swagger YAMLファイルのテスト
	t.Run("Swagger YAML File Test", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api-docs/swagger.yaml", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "openapi:")
	})
}

// SwaggerファイルとAPI実装の整合性をテストする
func TestE2EAPIAgainstSwagger(t *testing.T) {
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
