//go:build swagger_integration
// +build swagger_integration

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// テストデータベースコンテナを起動して接続するヘルパー関数
func setupTestDatabase(t *testing.T) (Storer, func()) {
	ctx := context.Background()

	// 元の環境変数を保存
	originalUser := os.Getenv("MYSQL_USER")
	originalPass := os.Getenv("MYSQL_PASSWORD")
	originalHost := os.Getenv("DB_HOST")
	originalPort := os.Getenv("DB_PORT")
	originalDB := os.Getenv("MYSQL_DATABASE")

	// MySQL初期化スクリプト（テーブル作成とテストデータ挿入）
	initScript := `
    CREATE TABLE IF NOT EXISTS stocks (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL UNIQUE,
        amount INT NOT NULL DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    );
    
    INSERT INTO stocks (name, amount) VALUES 
    ('apple', 100),
    ('banana', 200),
    ('orange', 150),
    ('grape', 75);
    `

	// 一時ファイルにSQLスクリプトを書き込む
	tmpFile, err := ioutil.TempFile("", "init-*.sql")
	if err != nil {
		t.Fatalf("一時ファイル作成エラー: %v", err)
	}
	if _, err := tmpFile.Write([]byte(initScript)); err != nil {
		t.Fatalf("初期化スクリプト書き込みエラー: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// MySQLコンテナの設定
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "rootpass",
			"MYSQL_DATABASE":      "stock_db_test",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "testpass",
		},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(tmpFile.Name(), "/docker-entrypoint-initdb.d/init.sql"),
		),
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("3306/tcp"),
			wait.ForLog("ready for connections").WithOccurrence(2), // 2回目の「接続準備完了」メッセージを待つ
		).WithStartupTimeout(2 * time.Minute),
		Cmd: []string{
			"--default-authentication-plugin=mysql_native_password",
			"--character-set-server=utf8mb4",
			"--collation-server=utf8mb4_unicode_ci",
			"--innodb_buffer_pool_size=20M", // 小さい値に設定してリソースを節約
		},
	}

	// MySQLコンテナの起動
	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("MySQLコンテナの起動に失敗: %v", err)
	}

	// コンテナのホストとポートを取得
	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("ホスト取得エラー: %v", err)
	}

	port, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatalf("ポート取得エラー: %v", err)
	}

	// 環境変数を設定
	os.Setenv("MYSQL_USER", "testuser")
	os.Setenv("MYSQL_PASSWORD", "testpass")
	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("MYSQL_DATABASE", "stock_db_test")

	// コンテナ情報をログに出力
	containerID := mysqlContainer.GetContainerID()
	t.Logf("MySQL コンテナ ID: %s", containerID)
	t.Logf("MySQL ホスト: %s, ポート: %s", host, port.Port())

	// コンテナのログを取得して表示
	logs, _ := mysqlContainer.Logs(ctx)
	defer logs.Close()
	logBytes, _ := ioutil.ReadAll(logs)
	t.Logf("コンテナログ: %s", string(logBytes))

	// DBへの接続が確立するまで待機
	var db *sql.DB
	maxRetries := 30
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// 接続を試行
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			"testuser", "testpass", host, port.Port(), "stock_db_test")
		db, err = sql.Open("mysql", dsn)

		if err == nil {
			err = db.Ping()
			if err == nil {
				t.Logf("テストデータベースに接続しました (host: %s, port: %s)", host, port.Port())
				break
			}
		}

		t.Logf("DB接続待機中... (%d/%d): %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}

	if err != nil {
		mysqlContainer.Terminate(ctx)
		t.Fatalf("データベースに接続できませんでした: %v", err)
	}

	// クリーンアップ関数
	cleanup := func() {
		if db != nil {
			db.Close()
		}
		mysqlContainer.Terminate(ctx)

		// 元の環境変数を復元
		os.Setenv("MYSQL_USER", originalUser)
		os.Setenv("MYSQL_PASSWORD", originalPass)
		os.Setenv("DB_HOST", originalHost)
		os.Setenv("DB_PORT", originalPort)
		os.Setenv("MYSQL_DATABASE", originalDB)
	}

	// SQLDB構造体をラップして返す
	return &SQLDB{DB: db}, cleanup
}
