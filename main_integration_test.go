//go:build integration
// +build integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ユーティリティ関数
func TestMain(m *testing.M) {
	// テスト環境の初期化
	code := m.Run()
	// テスト環境のクリーンアップ
	os.Exit(code)
}

// MySQLコンテナを起動し、接続用のSQLDBを返す関数
func setupMySQLContainer(t *testing.T) (*SQLDB, func(), error) {
	ctx := context.Background()

	// MySQLコンテナの設定
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.4.4",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections. Version: '8").
				WithStartupTimeout(time.Second*180)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("MySQLコンテナの起動に失敗しました: %v", err)
	}

	// コンテナのクリーンアップ関数
	cleanup := func() {
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Fatalf("MySQLコンテナの終了に失敗しました: %v", err)
		}
	}

	// データベース接続情報を取得
	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("ホスト名の取得に失敗しました: %v", err)
	}

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("ポートの取得に失敗しました: %v", err)
	}

	// データベース接続文字列
	dsn := fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb?parseTime=true", host, port.Port())

	// データベース接続
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("データベース接続に失敗しました: %v", err)
	}

	// 接続テスト（リトライ付き）
	var pingErr error
	for retries := 0; retries < 15; retries++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		t.Logf("Ping retry %d failed: %v", retries, pingErr)
		time.Sleep(2 * time.Second)
	}

	if pingErr != nil {
		cleanup()
		return nil, nil, fmt.Errorf("データベース接続テストに失敗しました: %v", pingErr)
	}

	// テーブル作成
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS stocks (
            name VARCHAR(255) PRIMARY KEY,
            amount INT NOT NULL
        )
    `)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("テーブル作成に失敗しました: %v", err)
	}

	return &SQLDB{DB: db}, cleanup, nil
}

// 結合テスト: API全体のテスト
func TestAPIIntegration(t *testing.T) {
	// テスト用ルーターの設定
	gin.SetMode(gin.TestMode)

	// MySQLコンテナのセットアップ
	db, cleanup, err := setupMySQLContainer(t)
	require.NoError(t, err)
	defer cleanup()
	defer db.Close()

	// テストデータを準備
	_, err = db.Exec("DELETE FROM stocks") // テーブルをクリア
	require.NoError(t, err)

	// Ginルーターのセットアップ
	router := gin.Default()
	setupRoutes(router, db)

	// テストケース
	t.Run("在庫の作成と取得", func(t *testing.T) {
		// 1. 在庫を登録する
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/stocks",
			newJSONReader(`{"name":"apple","amount":5}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"apple"`)
		assert.Contains(t, w.Body.String(), `"amount":5`)

		// 2. 登録した在庫を取得する
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/stocks/apple", nil)
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"apple"`)
		assert.Contains(t, w.Body.String(), `"amount":5`)

		// 3. 全ての在庫を取得する
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/stocks", nil)
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"apple"`)
		assert.Contains(t, w.Body.String(), `"amount":5`)

		t.Logf("body: %s", w.Body.String())

		t.Logf("ステータスコード - 期待値: %d, 実際: %d", http.StatusOK, w.Code)
	})

	t.Run("存在しない在庫の取得", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/stocks/nonexistent", nil)
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "データが存在しません")

		t.Logf("body: %s", w.Body.String())
	})

	t.Run("amountが未指定の場合はデフォルト値1が設定される", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/stocks",
			newJSONReader(`{"name":"banana"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"banana"`)
		assert.Contains(t, w.Body.String(), `"amount":1`)

		// 登録された値を確認
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/stocks/banana", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"banana"`)
		assert.Contains(t, w.Body.String(), `"amount":1`)

		// 3. 全ての在庫を取得する
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/stocks", nil)
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		// assert.Contains(t, w.Body.String(), `"name":"apple"`)
		// assert.Contains(t, w.Body.String(), `"amount":5`)

		// // Use JSONEq for exact JSON comparison instead of Contains
		// expectedJSON := `{"stocks":[{"name":"apple","amount":5},{"name":"banana","amount":1}]}`

		// assert.JSONEq(t, expectedJSON, w.Body.String())

		// t.Logf("body: %s", w.Body.String())

		// t.Logf("ステータスコード - 期待値: %d, 実際: %d", http.StatusOK, w.Code)

	})
	t.Run("appleに3000の在庫をいれて、全部の在庫を確認", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/stocks",
			newJSONReader(`{"name":"apple","amount":3000}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"name":"apple"`)
		assert.Contains(t, w.Body.String(), `"amount":3005`)

		// 3. 全ての在庫を取得する
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/stocks", nil)
		router.ServeHTTP(w, req)

		// レスポンスを検証
		assert.Equal(t, http.StatusOK, w.Code)
		// レスポンスをJSONとしてパース
		var stocks []Stock
		err = json.Unmarshal(w.Body.Bytes(), &stocks)
		require.NoError(t, err, "レスポンスのJSONパースに失敗しました")

		// スライスの長さを確認
		assert.Len(t, stocks, 2, "在庫アイテムは2つあるべきです")

		// 個別の要素を確認
		if len(stocks) >= 2 {
			// 要素の順序が不定の場合に対応
			var appleFound, bananaFound bool

			for _, stock := range stocks {
				if stock.Name == "apple" {
					assert.Equal(t, 3005, stock.Amount, "appleの在庫量が正しくありません")
					appleFound = true
				}
				if stock.Name == "banana" {
					assert.Equal(t, 1, stock.Amount, "bananaの在庫量が正しくありません")
					bananaFound = true
				}
			}

			assert.True(t, appleFound, "appleの在庫が見つかりませんでした")
			assert.True(t, bananaFound, "bananaの在庫が見つかりませんでした")
		}

		t.Logf("body: %s", w.Body.String())

		// t.Logf("ステータスコード - 期待値: %d, 実際: %d", http.StatusOK, w.Code)

	})
}

// JSONリクエストボディを作成するヘルパー関数
func newJSONReader(jsonStr string) *strings.Reader {
	return strings.NewReader(jsonStr)
}
