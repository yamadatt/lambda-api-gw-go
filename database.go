package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Storer is an interface for accessing the database.
type Storer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Close() error // Close メソッドを追加

}

// SQLDB は sql.DB をラップし、Storer インターフェースを実装します。
type SQLDB struct {
	DB *sql.DB
}

func (s *SQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.DB.Exec(query, args...)
}

func (s *SQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.DB.QueryRow(query, args...)
}

func (s *SQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.DB.Query(query, args...)
}

// Close メソッドを実装
func (s *SQLDB) Close() error {
	return s.DB.Close()
}

// sql.Open の代わりに呼び出す関数（グローバル変数として定義）
var sqlOpenFunc = sql.Open

// グローバル変数でDB接続を保持
var (
	dbConn     Storer
	dbConnOnce sync.Once
	dbConnErr  error
)

// Lambda向けにデータベース接続を最適化
func connectDB() (Storer, error) {
	// 一度だけDB接続を行い、その後は再利用
	dbConnOnce.Do(func() {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Warning: .env file not found, using environment variables")
		}

		// 環境変数から読み込み
		dbUser := getEnv("MYSQL_USER", "root")
		dbPassword := getEnv("MYSQL_PASSWORD", "root")
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "3306")
		dbName := getEnv("MYSQL_DATABASE", "stock_db")

		// Lambda用の接続設定（一般的な最適値）
		maxRetries := 3
		currentRetry := 0

		for currentRetry < maxRetries {
			// 接続文字列を構築
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
				dbUser, dbPassword, dbHost, dbPort, dbName)

			db, err := sqlOpenFunc("mysql", dsn)
			if err == nil {
				// 接続テスト
				err = db.Ping()
				if err == nil {
					// Lambda向けに接続プールを最適化
					db.SetMaxOpenConns(5)
					db.SetMaxIdleConns(5)
					db.SetConnMaxLifetime(5 * time.Minute)

					log.Printf("Successfully connected to database %s on %s", dbName, dbHost)
					dbConn = &SQLDB{DB: db}
					return
				}
			}

			log.Printf("Failed to connect to database (attempt %d/%d): %v",
				currentRetry+1, maxRetries, err)
			currentRetry++
			time.Sleep(1 * time.Second)
		}

		dbConnErr = fmt.Errorf("failed to connect to database after %d attempts", maxRetries)
	})

	return dbConn, dbConnErr
}

// 環境変数取得ヘルパー
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
