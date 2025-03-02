//go:build swagger
// +build swagger

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Swagger定義の簡易構造体
type SwaggerDefinition struct {
	Swagger string                 `yaml:"swagger"`
	Info    map[string]interface{} `yaml:"info"`
	Paths   map[string]struct {
		Get    map[string]interface{} `yaml:"get"`
		Post   map[string]interface{} `yaml:"post"`
		Put    map[string]interface{} `yaml:"put"`
		Delete map[string]interface{} `yaml:"delete"`
	} `yaml:"paths"`
}

func TestSwaggerDefinitionAgainstAPI(t *testing.T) {
	t.Log("Swaggerテストを実行しています...")

	if testing.Short() {
		t.Skip("詳細なSwagger検証はshortモードではスキップされます")
	}

	// テスト用のDB設定ファイルを作成
	tmpFile, err := ioutil.TempFile("", "db_config.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 設定を書き込む
	configData := []byte(`{"host": "192.168.1.49", "port": 3306, "user": "your_db_user", "password": "your_db_password", "database": "your_db_name"}`)
	_, err = tmpFile.Write(configData)
	require.NoError(t, err)
	tmpFile.Close()

	// 環境変数で設定ファイルパスを変更
	os.Setenv("DB_CONFIG_PATH", tmpFile.Name())

	// 1. swagger.yamlファイルを読み込み
	yamlFile, err := ioutil.ReadFile("./swagger.yaml")
	require.NoError(t, err, "swagger.yamlの読み込みに失敗しました")

	// 2. YAML解析
	var swaggerDef SwaggerDefinition
	err = yaml.Unmarshal(yamlFile, &swaggerDef)
	require.NoError(t, err, "swagger.yamlの解析に失敗しました")

	// 3. DB接続
	db, err := connectDB()
	if err != nil {
		t.Skipf("データベース接続に失敗したためテストをスキップします: %v", err)
	}

	// 4. ルーター設定
	gin.SetMode(gin.TestMode)
	r := gin.New()
	setupRoutes(r, db)

	// 5. Swagger定義からすべてのエンドポイントをテスト
	for path, methods := range swaggerDef.Paths {
		// GETメソッドのテスト
		if methods.Get != nil {
			t.Run("GET "+path, func(t *testing.T) {
				// パスパラメータの置換とバージョンプレフィックスの追加
				testPath := path
				if !strings.HasPrefix(path, "/v1") {
					testPath = "/v1" + path
				}
				if strings.Contains(path, "{") {
					testPath = strings.Replace(testPath, "{name}", "test", -1)
					testPath = strings.Replace(testPath, "{id}", "1", -1)
				}

				req := httptest.NewRequest("GET", testPath, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				// ステータスコードの検証
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound,
					"GETエンドポイント %s の応答コードが想定外です: %d", testPath, w.Code)

				// レスポンスがJSON形式であることを検証
				contentType := w.Header().Get("Content-Type")
				if w.Code == http.StatusOK {
					assert.Contains(t, contentType, "application/json")

					// JSONパース検証
					var response interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					assert.NoError(t, err, "JSONパースに失敗しました: %s", testPath)
				}
			})
		}

		// POSTメソッドのテスト
		if methods.Post != nil {
			t.Run("POST "+path, func(t *testing.T) {
				// リクエストボディの準備
				requestBody := `{"name": "test_item", "amount": 42}`

				// パスの調整（必要に応じてバージョンプレフィックスを追加）
				testPath := path
				if !strings.HasPrefix(testPath, "/v1") {
					testPath = "/v1" + testPath
				}

				req := httptest.NewRequest("POST", testPath, strings.NewReader(requestBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				// 応答コードのチェック
				assert.True(t,
					w.Code == http.StatusOK ||
						w.Code == http.StatusCreated ||
						w.Code == http.StatusBadRequest,
					"POSTエンドポイント %s の応答コードが想定外です: %d", testPath, w.Code)
			})
		}

		// 他のHTTPメソッドも同様にテスト...
	}
}
