package main

import (
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestConnectDB(t *testing.T) {
	// CI環境の場合はモックを使用
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		// モックテスト
		originalSqlOpenFunc := sqlOpenFunc
		defer func() { sqlOpenFunc = originalSqlOpenFunc }()

		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			db, mock, _ := sqlmock.New()
			mock.ExpectPing()
			return db, nil
		}

		db, err := connectDB()
		assert.NoError(t, err)
		assert.NotNil(t, db)
		return
	}

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

	t.Run("Successful database connection", func(t *testing.T) {
		// モックデータベースの作成
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("Failed to create mock database connection: %v", err)
		}

		// deferによるClose()は削除（手動でClose()するため）

		// モックの設定 (Ping と Close の成功を期待)
		mock.ExpectPing()
		mock.ExpectClose() // 明示的にClose期待値を追加

		// sqlOpenFuncをモック関数で置き換え
		originalSqlOpenFunc := sqlOpenFunc
		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { sqlOpenFunc = originalSqlOpenFunc }()

		// connectDB関数を呼び出し
		storer, err := connectDB()
		assert.NoError(t, err)
		assert.NotNil(t, storer)

		// Storer が Close() メソッドを持っていることを確認して呼び出す
		err = storer.Close()
		assert.NoError(t, err)

		// すべての期待が満たされたことを確認
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	// Restore original environment variables
	os.Setenv("MYSQL_USER", originalUser)
	os.Setenv("MYSQL_PASSWORD", originalPass)
	os.Setenv("DB_HOST", originalHost)
	os.Setenv("MYSQL_DATABASE", originalDB)
}
