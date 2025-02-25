package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

func connectDB() (Storer, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	fmt.Println("Connecting to database")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("MYSQL_DATABASE"),
	)

	db, err := sqlOpenFunc("mysql", dsn)  // sql.Open の代わりに sqlOpenFunc を使用
	fmt.Println("log:sqlOpenFunc called") // ログを追加
	if err != nil {
		fmt.Println("Error connecting to database", dsn)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("Error pinging database", err)
		return nil, err
	}

	fmt.Println("Database connection established")
	return &SQLDB{DB: db}, nil // SQLDB を返す
}
