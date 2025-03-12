// dbmock_test.go
package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// NewMockSQLDB は sqlmock を使ってモック用の *sql.DB と、それをラップした SQLDB インスタンス、さらに sqlmock.Sqlmock を返します。
// テスト中にエラーがあれば t.Fatalf() で中断します。
func NewMockSQLDB(t *testing.T) (*SQLDB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	// Ping の期待値をセット（必要に応じてテスト側で設定してもよい）
	mock.ExpectPing()
	// Ping を実際に呼んでおくと、モック側で期待値チェックが走ります
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping mock db: %v", err)
	}
	return &SQLDB{DB: db}, mock
}

// NewMockSQLDBWithExpectations は sqlmock を使ってモック用の *sql.DB と、それをラップした SQLDB インスタンス、さらに sqlmock.Sqlmock を返します。
// 標準的な期待値を設定します。
func NewMockSQLDBWithExpectations(t *testing.T) (*SQLDB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database connection: %v", err)
	}

	// 標準的な期待値を設定
	mock.ExpectPing()
	mock.ExpectClose()

	return &SQLDB{DB: db}, mock
}

// NewMockDB は sqlmock を使ってモック用の DB と sqlmock のインスタンスを返すヘルパー関数です。
// テストで必ずエラーがない状態でモックが作成できることを保証します。
func NewMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	return db, mock
}

// SetupSQLOpenFunc は sqlOpenFunc をモック用の関数に置き換えるためのヘルパーです。
// 引数としてモックの *sql.DB を受け取り、呼び出し時にその DB を返す関数を返します。
func SetupSQLOpenFunc(db *sql.DB) func(driverName, dataSourceName string) (*sql.DB, error) {
	return func(driverName, dataSourceName string) (*sql.DB, error) {
		return db, nil
	}
}
