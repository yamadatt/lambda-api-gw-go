//go:build oapi
// +build oapi

package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPISchemaValidity(t *testing.T) {
	// OpenAPIスキーマをロード
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("swagger.yaml")
	require.NoError(t, err, "OpenAPIスキーマをロードできませんでした")

	// スキーマの検証
	err = doc.Validate(loader.Context)
	assert.NoError(t, err, "OpenAPIスキーマが無効です")
}

func TestRoutesMappingToOpenAPI(t *testing.T) {
	// GinルーターをセットアップしてルートをMAP
	router := setupRouter()

	// 一時的なサーバーを起動してAPIハンドラーを登録
	store := &MockStore{} // モックストアの実装が必要
	setupRoutes(router, store)

	// 登録されたルートを取得
	routes := router.Routes()

	// OpenAPIスキーマをロード
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("swagger.yaml")
	require.NoError(t, err)

	// 各ルートがOpenAPIに定義されていることを確認
	for _, route := range routes {
		// システム関連のパス(/swaggerなど）はスキップ
		if isSystemPath(route.Path) {
			continue
		}

		// パスとメソッドの組み合わせがOpenAPIに存在するか確認
		path := convertGinPathToOpenAPI(route.Path)
		pathItem := doc.Paths.Find(path)

		assert.NotNilf(t, pathItem, "パス '%s' がOpenAPIスキーマに定義されていません", path)
		if pathItem != nil {
			// HTTPメソッドに対応する操作が定義されているか確認
			op := getOperationForMethod(pathItem, route.Method)
			assert.NotNilf(t, op, "パス '%s' のメソッド '%s' がOpenAPIスキーマに定義されていません",
				path, route.Method)
		}
	}
}

// ヘルパー関数
func isSystemPath(path string) bool {
	// Swagger UI, ヘルスチェックなどのシステムパスをフィルタリング
	systemPaths := []string{"/swagger", "/health", "/metrics", "/api-docs"}
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) {
			return true
		}
	}
	return false
}

func convertGinPathToOpenAPI(ginPath string) string {
	// Ginのパスパラメータ形式(:name)をOpenAPIの形式({name})に変換
	re := regexp.MustCompile("/:([^/]+)")
	return re.ReplaceAllString(ginPath, "/{$1}")
}

func getOperationForMethod(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	switch method {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "DELETE":
		return pathItem.Delete
	case "PATCH":
		return pathItem.Patch
	case "HEAD":
		return pathItem.Head
	case "OPTIONS":
		return pathItem.Options
	case "TRACE":
		return pathItem.Trace
	}
	return nil
}
