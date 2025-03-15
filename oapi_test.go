//go:build oapi

package main

import (
	"database/sql" // この行を追加
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Error variables
var ErrStockNotFound = errors.New("stock not found")

// テスト用のAppConfigを作成するヘルパー関数
func testAppConfig() AppConfig {
	return AppConfig{
		Host: "localhost",
		Port: "8080",
		// その他の設定は実際のAppConfig構造体に存在しないため削除
	}
}

// MockStore は Storer インターフェースのテスト実装
type MockStore struct {
	Calls map[string]int
	Data  map[string]Stock
}

func NewMockStore() *MockStore {
	return &MockStore{
		Calls: make(map[string]int),
		Data: map[string]Stock{
			"apple":  {Name: "apple", Amount: 100},
			"banana": {Name: "banana", Amount: 200},
		},
	}
}

func (m *MockStore) GetAllStocks() ([]Stock, error) {
	m.Calls["GetAllStocks"]++
	stocks := make([]Stock, 0, len(m.Data))
	for _, s := range m.Data {
		stocks = append(stocks, s)
	}
	return stocks, nil
}

// GetStock は特定の在庫を返すモックメソッド
func (m *MockStore) GetStock(name string) (*Stock, error) {
	m.Calls["GetStock"]++
	if stock, ok := m.Data[name]; ok {
		return &stock, nil
	}
	return nil, ErrStockNotFound
}

// UpdateStock は在庫を更新するモックメソッド
func (m *MockStore) UpdateStock(stock Stock) error {
	m.Calls["UpdateStock"]++
	m.Data[stock.Name] = stock
	return nil
}

// CreateStock は在庫を作成するモックメソッド
func (m *MockStore) CreateStock(stock Stock) error {
	m.Calls["CreateStock"]++
	m.Data[stock.Name] = stock
	return nil
}

// Close は接続を閉じるモックメソッド
func (m *MockStore) Close() error {
	m.Calls["Close"]++
	return nil
}

// Exec はクエリを実行するモックメソッド
func (m *MockStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.Calls["Exec"]++
	return &MockResult{}, nil
}

// MockResult はsql.Resultインターフェースを実装
type MockResult struct{}

func (m *MockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (m *MockResult) RowsAffected() (int64, error) {
	return 1, nil
}

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
	// テスト用のAppConfigを取得
	config := testAppConfig()

	// GinルーターをセットアップしてルートをMAP
	router := setupRouter(config)

	// 一時的なサーバーを起動してAPIハンドラーを登録
	store := NewMockStore()
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

func TestMain(m *testing.M) {
	// テスト全体の前処理
	code := m.Run()
	// テスト全体の後処理
	os.Exit(code)
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
