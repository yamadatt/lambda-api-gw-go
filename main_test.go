package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetupRoutes(t *testing.T) {
	// Original environment variables
	originalUser := os.Getenv("MYSQL_USER")
	originalPass := os.Getenv("MYSQL_PASSWORD")
	originalHost := os.Getenv("DB_HOST")
	originalDB := os.Getenv("MYSQL_DATABASE")

	// 環境変数を一時的に削除
	os.Unsetenv("MYSQL_USER")
	os.Unsetenv("MYSQL_PASSWORD")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("MYSQL_DATABASE")

	defer func() {
		// 環境変数を復元
		os.Setenv("MYSQL_USER", originalUser)
		os.Setenv("MYSQL_PASSWORD", originalPass)
		os.Setenv("DB_HOST", originalHost)
		os.Setenv("MYSQL_DATABASE", originalDB)
	}()

	// モックデータベースの作成
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database connection: %v", err)
	}
	defer db.Close()

	// モックの設定 (Ping が成功することを期待)
	mock.ExpectPing()

	// モックデータベースを使用する SQLDB を直接作成
	mockStorer := &SQLDB{DB: db}

	// connectDB 関数の代わりに直接モックを使用
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	setupRoutes(router, mockStorer)

	// モックの準備：getAllStocks 用のクエリ設定
	mock.ExpectQuery("SELECT \\* FROM stocks").WillReturnRows(
		sqlmock.NewRows([]string{"name", "amount"}).
			AddRow("test_stock", 10))

	t.Run("GET /v1/stocks returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/stocks", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// すべての期待が満たされたことを確認
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
