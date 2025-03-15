package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// モックハンドラー実装
type MockStockHandler struct{}

func (m *MockStockHandler) GetAllStocks(c *gin.Context) {
	stocks := []Stock{
		{Name: "apple", Amount: intPtr(100)},
		{Name: "banana", Amount: intPtr(200)},
	}
	c.JSON(http.StatusOK, stocks)
}

func (m *MockStockHandler) CreateOrUpdateStock(c *gin.Context) {
	var stock StockRequest
	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "無効なリクエスト"})
		return
	}

	result := Stock{Name: stock.Name, Amount: stock.Amount}
	c.JSON(http.StatusOK, result)
}

func (m *MockStockHandler) GetStockByName(c *gin.Context, name string) {
	// テスト用のモックデータ
	stockData := map[string]Stock{
		"apple":  {Name: "apple", Amount: intPtr(100)},
		"banana": {Name: "banana", Amount: intPtr(200)},
	}

	if stock, ok := stockData[name]; ok {
		c.JSON(http.StatusOK, stock)
		return
	}

	c.JSON(http.StatusNotFound, ErrorResponse{Error: "在庫が見つかりません"})
}

// テストヘルパー関数
func intPtr(i int) *int {
	return &i
}

func setupTestRouter() (*gin.Engine, *httptest.Server) {
	// テストモードに設定
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := &MockStockHandler{}
	RegisterHandlers(router, handler)

	ts := httptest.NewServer(router)
	return router, ts
}

// テストケース
func TestGetAllStocks(t *testing.T) {
	_, ts := setupTestRouter()
	defer ts.Close()

	// クライアント作成
	client, err := NewClientWithResponses(ts.URL)
	assert.NoError(t, err)

	// APIリクエスト
	resp, err := client.GetAllStocksWithResponse(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	// レスポンス本文の検証
	var stocks []Stock
	err = json.Unmarshal(resp.Body, &stocks)
	assert.NoError(t, err)
	assert.Len(t, stocks, 2)
	assert.Equal(t, "apple", stocks[0].Name)
	assert.Equal(t, 100, *stocks[0].Amount)
}

func TestCreateOrUpdateStock(t *testing.T) {
	_, ts := setupTestRouter()
	defer ts.Close()

	client, err := NewClientWithResponses(ts.URL)
	assert.NoError(t, err)

	// リクエスト本文
	amount := 150
	requestBody := StockRequest{
		Name:   "orange",
		Amount: &amount,
	}

	// APIリクエスト送信
	resp, err := client.CreateOrUpdateStockWithResponse(context.Background(), requestBody)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	// レスポンスのJSON検証
	assert.NotNil(t, resp.JSON200)
	assert.Equal(t, "orange", resp.JSON200.Name)
	assert.Equal(t, amount, *resp.JSON200.Amount)
}

func TestGetStockByName(t *testing.T) {
	_, ts := setupTestRouter()
	defer ts.Close()

	client, err := NewClientWithResponses(ts.URL)
	assert.NoError(t, err)

	// 存在する商品のテスト
	resp, err := client.GetStockByNameWithResponse(context.Background(), "apple")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	// レスポンス本文の検証
	var stock Stock
	err = json.Unmarshal(resp.Body, &stock)
	assert.NoError(t, err)
	assert.Equal(t, "apple", stock.Name)
	assert.Equal(t, 100, *stock.Amount)

	// 存在しない商品のテスト
	resp2, err := client.GetStockByNameWithResponse(context.Background(), "not-exists")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode())
}
