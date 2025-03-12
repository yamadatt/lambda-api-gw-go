//go:build swagger_integration
// +build swagger_integration

package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// テストデータベースコンテナを起動して接続するヘルパー関数
func setupTestDatabase(t *testing.T) (Storer, func()) {
	// docker-compose.ymlが存在するか確認
	if _, err := os.Stat("docker-compose.yaml"); os.IsNotExist(err) {
		t.Fatalf("docker-compose.ymlファイルが見つかりません。プロジェクトのルートディレクトリに配置してください")
	}

	// 元の環境変数を保存
	originalUser := os.Getenv("MYSQL_USER")
	originalPass := os.Getenv("MYSQL_PASSWORD")
	originalHost := os.Getenv("DB_HOST")
	originalPort := os.Getenv("DB_PORT")
	originalDB := os.Getenv("MYSQL_DATABASE")

	// テスト用の環境変数を設定
	os.Setenv("MYSQL_USER", "your_user")
	os.Setenv("MYSQL_PASSWORD", "your_password")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("MYSQL_DATABASE", "your_database")

	// Docker Composeでコンテナを起動
	cmd := exec.Command("docker-compose", "up", "-d", "mysql")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker Composeの起動に失敗しました: %v\n出力: %s", err, output)
	}

	// データベースが準備できるまで待機
	time.Sleep(5 * time.Second)

	// データベースに接続
	db, err := connectDB()
	if err != nil {
		t.Fatalf("データベース接続に失敗しました: %v", err)
	}
	// デバッグ用：接続情報を表示
	t.Logf("Database connection established:")
	t.Logf("User: %s", os.Getenv("MYSQL_USER"))
	t.Logf("Host: %s", os.Getenv("DB_HOST"))
	t.Logf("Port: %s", os.Getenv("DB_PORT"))
	t.Logf("Database: %s", os.Getenv("MYSQL_DATABASE"))

	//dbにpingを送る
	// if err := db.Ping(); err != nil {
	// 	t.Fatalf("データベースへのpingに失敗しました: %v", err)
	// }

	// 初期データを投入（オプション）
	setupSQL := `
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
    ('orange', 150)
    ON DUPLICATE KEY UPDATE amount = VALUES(amount);
    `

	// SQLファイルを一時的に作成
	tmpfile, err := ioutil.TempFile("", "setup.*.sql")
	if err != nil {
		t.Fatalf("一時SQLファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(setupSQL)); err != nil {
		t.Fatalf("SQLファイルの書き込みに失敗しました: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("SQLファイルのクローズに失敗しました: %v", err)
	}

	// SQLファイルを実行
	sqlCmd := exec.Command("docker-compose", "exec", "-T", "mysql", "mysql", "-u"+os.Getenv("MYSQL_USER"), "-p"+os.Getenv("MYSQL_PASSWORD"), os.Getenv("MYSQL_DATABASE"))
	sqlCmd.Stdin, err = os.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("SQLファイルを開けませんでした: %v", err)
	}
	sqlOutput, err := sqlCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SQLの実行に失敗しました: %v\n出力: %s", err, sqlOutput)
	}

	// データベースに接続
	db, err = connectDB()
	if err != nil {
		t.Fatalf("データベース接続に失敗しました: %v", err)
	}
	// デバッグ用：接続情報を表示
	t.Logf("Database connection established:")
	t.Logf("User: %s", os.Getenv("MYSQL_USER"))
	t.Logf("Host: %s", os.Getenv("DB_HOST"))
	t.Logf("Port: %s", os.Getenv("DB_PORT"))
	t.Logf("Database: %s", os.Getenv("MYSQL_DATABASE"))
	// クリーンアップ関数を返す
	cleanup := func() {
		// 環境変数を復元
		os.Setenv("MYSQL_USER", originalUser)
		os.Setenv("MYSQL_PASSWORD", originalPass)
		os.Setenv("DB_HOST", originalHost)
		os.Setenv("DB_PORT", originalPort)
		os.Setenv("MYSQL_DATABASE", originalDB)

		// コンテナを停止（オプション）
		// 継続的にコンテナを動かしたい場合は、この行をコメントアウト
		stopCmd := exec.Command("docker-compose", "down")
		stopCmd.Run()
	}

	return db, cleanup
}
