//go:build e2e
// +build e2e

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func TestSwaggerEndpoint(t *testing.T) {
	// テスト用のリクエストを作成
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()

	// Ginエンジンでリクエストを処理
	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.ServeHTTP(w, req)

	// レスポンスの検証
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Swagger UI")
}
