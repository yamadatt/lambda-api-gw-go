//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"lambda-api-gw-go/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnvironment() (*api.ClientWithResponses, func(), error) {
	// テスト用のサーバーを起動
	config := AppConfig{
		Host: "localhost",
		Port: "8089", // テスト用ポート
		DBConfig: DBConfig{
			Host:     "localhost",
			Port:     "3306",
			User:     "test",
			Password: "test",
			Database: "test_db",
		},
	}

	// データベース接続
	db, err := connectDB()
	if err != nil {
		return nil, nil, fmt.Errorf("DBコネクション作成失敗: %v", err)
	}

	// テストデータをセットアップ
	err = setupTestData(db)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("テストデータ作成失敗: %v", err)
	}

	// サーバー起動
	router := setupRouter(config)
	setupRoutes(router, db)

	// 非同期でサーバー起動
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.Host, config.Port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("サーバー起動エラー: %v\n", err)
		}
	}()

	// サーバー起動待ち
	time.Sleep(100 * time.Millisecond)

	// クライアント作成
	client, err := api.NewClientWithResponses(fmt.Sprintf("http://%s:%s", config.Host, config.Port))
	if err != nil {
		server.Shutdown(context.Background())
		db.Close()
		return nil, nil, fmt.Errorf("クライアント作成失敗: %v", err)
	}

	// クリーンアップ関数
	cleanup := func() {
		server.Shutdown(context.Background())
		cleanupTestData(db)
		db.Close()
	}

	return client, cleanup, nil
}

func setupTestData(db Storer) error {
	// テストデータを作成
	stocks := []Stock{
		{Name: "apple", Amount: 100},
		{Name: "banana", Amount: 200},
	}

	for _, stock := range stocks {
		err := db.CreateStock(stock)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestData(db Storer) error {
	// テスト後のデータ削除
	_, err := db.Exec("DELETE FROM stocks")
	return err
}

func TestIntegrationGetAllStocks(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストはshortモードでスキップ")
	}

	client, cleanup, err := setupTestEnvironment()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	resp, err := client.GetAllStocksWithResponse(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	// 他の検証...
}

func TestIntegrationCreateAndGetStock(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストはshortモードでスキップ")
	}

	client, cleanup, err := setupTestEnvironment()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// 新しい在庫を作成
	amount := 150
	newStock := api.StockRequest{
		Name:   "orange",
		Amount: &amount,
	}

	createResp, err := client.CreateOrUpdateStockWithResponse(ctx, newStock)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, createResp.StatusCode())

	// 作成した在庫を取得
	getResp, err := client.GetStockByNameWithResponse(ctx, "orange")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getResp.StatusCode())

	// 内容の検証
	var retrievedStock api.Stock
	err = json.Unmarshal(getResp.Body, &retrievedStock)
	require.NoError(t, err)
	assert.Equal(t, "orange", retrievedStock.Name)
	assert.Equal(t, amount, *retrievedStock.Amount)
}
