package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetStocksHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		stockName    string
		mockSetup    func(mock sqlmock.Sqlmock)
		expectedCode int
		expectedBody string
	}{
		{
			name:      "データが存在しない場合",
			stockName: "apple",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM stocks WHERE name = \\?").
					WithArgs("apple").
					WillReturnRows(sqlmock.NewRows([]string{"name", "amount"}))
			},
			expectedCode: http.StatusOK,
			expectedBody: "データが存在しません",
		},
		{
			name:      "データが存在する場合",
			stockName: "banana",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM stocks WHERE name = \\?").
					WithArgs("banana").
					WillReturnRows(sqlmock.NewRows([]string{"name", "amount"}).
						AddRow("banana", 10))
			},
			expectedCode: http.StatusOK,
			expectedBody: `"name":"banana","amount":10`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// モックデータベースのセットアップ
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock database: %v", err)
			}
			defer db.Close()

			// モックStorerを作成
			mockStorer := &SQLDB{DB: db}

			// モックの設定
			tc.mockSetup(mock)

			router := gin.Default()
			router.GET("/stocks/:name", getStocksHandler(mockStorer))

			req, _ := http.NewRequest(http.MethodGet, "/stocks/"+tc.stockName, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// レスポンスボディの内容を標準出力に出力

			responseBody := w.Body.String()
			t.Logf("テストケース: %s", tc.name)
			t.Logf("レスポンスボディ: %s", responseBody)
			t.Logf("ステータスコード - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)

			// ステータスコードの検証
			if !assert.Equal(t, tc.expectedCode, w.Code) {
				t.Errorf("ステータスコードが一致しません - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)
			}

			// レスポンスボディの検証
			if !assert.Contains(t, responseBody, tc.expectedBody) {
				t.Errorf("レスポンスボディに期待する文字列が含まれていません")
				t.Errorf("期待値: %s", tc.expectedBody)
				t.Errorf("実際の値: %s", responseBody)
			}

			// モック期待値の確認
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("未処理の期待値があります: %s", err)
			}
		})
	}
}
func TestGetAllStocksHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		mockSetup    func(mock sqlmock.Sqlmock)
		expectedCode int
		expectedBody string
	}{
		{
			name: "データが存在しない場合",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM stocks").
					WillReturnRows(sqlmock.NewRows([]string{"name", "amount"}))
			},
			expectedCode: http.StatusOK,
			expectedBody: "データが存在しません",
		},
		// 必要に応じて他のケースを追加
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// モックデータベースのセットアップ
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock database: %v", err)
			}
			defer db.Close()

			// モックStorerを作成
			mockStorer := &SQLDB{DB: db}

			// モックの設定
			tc.mockSetup(mock)

			router := gin.Default()
			router.GET("/stocks", getAllStocksHandler(mockStorer))

			req, _ := http.NewRequest(http.MethodGet, "/stocks", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// レスポンスの詳細をログに記録
			responseBody := w.Body.String()
			t.Logf("テストケース: %s", tc.name)
			t.Logf("レスポンスボディ: %s", responseBody)
			t.Logf("ステータスコード - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)

			// ステータスコードの検証
			if !assert.Equal(t, tc.expectedCode, w.Code) {
				t.Errorf("ステータスコードが一致しません - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)
			}

			// レスポンスボディの検証
			if !assert.Contains(t, responseBody, tc.expectedBody) {
				t.Errorf("レスポンスボディに期待する文字列が含まれていません")
				t.Errorf("期待値: %s", tc.expectedBody)
				t.Errorf("実際の値: %s", responseBody)
			}

			// モック期待値の確認
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("未処理の期待値があります: %s", err)
			}
		})
	}
}

func TestPostStocksHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		requestBody  string
		mockSetup    func(mock sqlmock.Sqlmock)
		expectedCode int
		expectedBody string
	}{
		{
			name:        "正常な登録",
			requestBody: `{"name":"banana","amount":10}`,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO stocks").
					WithArgs("banana", 10, 10).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectQuery("SELECT \\* FROM stocks WHERE name = \\?").
					WithArgs("banana").
					WillReturnRows(sqlmock.NewRows([]string{"name", "amount"}).
						AddRow("banana", 10))
			},
			expectedCode: http.StatusOK,
			expectedBody: `"name":"banana"`,
		},
		{
			name:        "amountが未指定の場合はデフォルト値1が設定される",
			requestBody: `{"name":"apple"}`,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// amountに1がセットされていることを検証
				mock.ExpectExec("INSERT INTO stocks").
					WithArgs("apple", 1, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectQuery("SELECT \\* FROM stocks WHERE name = \\?").
					WithArgs("apple").
					WillReturnRows(sqlmock.NewRows([]string{"name", "amount"}).
						AddRow("apple", 1))
			},
			expectedCode: http.StatusOK,
			expectedBody: `"name":"apple","amount":1`,
		},
		// 必要に応じて他のケースを追加
		// 例: バリデーションエラー、データベースエラーなど
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// モックデータベースのセットアップ
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock database: %v", err)
			}
			defer db.Close()

			// モックStorerを作成
			mockStorer := &SQLDB{DB: db}

			// モックの設定
			tc.mockSetup(mock)

			router := gin.Default()
			router.POST("/stocks", postStocksHandler(mockStorer))

			body := bytes.NewBufferString(tc.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/stocks", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// レスポンスの詳細をログに記録
			responseBody := w.Body.String()
			t.Logf("テストケース: %s", tc.name)
			t.Logf("リクエストボディ: %s", tc.requestBody)
			t.Logf("レスポンスボディ: %s", responseBody)
			t.Logf("ステータスコード - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)

			// ステータスコードの検証
			if !assert.Equal(t, tc.expectedCode, w.Code) {
				t.Errorf("ステータスコードが一致しません - 期待値: %d, 実際: %d", tc.expectedCode, w.Code)
			}

			// レスポンスボディの検証（期待値が設定されている場合）
			if tc.expectedBody != "" {
				if !assert.Contains(t, responseBody, tc.expectedBody) {
					t.Errorf("レスポンスボディに期待する文字列が含まれていません")
					t.Errorf("期待値: %s", tc.expectedBody)
					t.Errorf("実際の値: %s", responseBody)
				}
			}

			// モック期待値の確認
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("未処理の期待値があります: %s", err)
			}
		})
	}
}
