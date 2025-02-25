package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StockRequest はリクエストボディの構造体です。

type Stock struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

// getStocksHandler は GET /stocks/:name のリクエストを処理します。
func getStocksHandler(db Storer) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		stocks, err := getStocks(db, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Println(stocks)
		if stocks == nil {
			// データが存在しない場合の処理
			// メッセージにデーが存在しませんと返す
			c.JSON(http.StatusOK, gin.H{"message": "データが存在しません"})

		} else {
			// データが存在する場合の処理

			c.JSON(http.StatusOK, stocks)
		}

	}
}

func getAllStocksHandler(db Storer) gin.HandlerFunc {
	return func(c *gin.Context) {
		stocks, err := getAllStocks(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Println(stocks)
		if stocks == nil {
			// データが存在しない場合の処理
			// メッセージにデーが存在しませんと返す
			c.JSON(http.StatusOK, gin.H{"message": "データが存在しません"})
		} else {
			// データが存在する場合の処理
			c.JSON(http.StatusOK, stocks)
		}

	}
}

func getStocks(db Storer, name string) ([]Stock, error) {
	rows, err := db.Query("SELECT * FROM stocks WHERE name = ?", name)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		fmt.Println("rows is nil after db.Query") // ログを追加
		return []Stock{}, nil                     // 空のスライスと nil エラーを返す
	}
	defer func() {
		if rows != nil {
			err := rows.Close()
			if err != nil {
				fmt.Println("Error closing rows:", err) // ログを追加
			}
		}
	}()

	var stocks []Stock
	for rows.Next() {
		var stock Stock
		if err := rows.Scan(&stock.Name, &stock.Amount); err != nil {
			return nil, err
		}
		stocks = append(stocks, stock)
	}
	return stocks, nil
}

// postStocksHandler は POST /stocks のリクエストを処理します。
func postStocksHandler(db Storer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stockReq Stock
		if err := c.ShouldBindJSON(&stockReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if stockReq.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
			return
		}

		if stockReq.Amount == 0 {
			stockReq.Amount = 1
		}
		fmt.Println(stockReq)
		err := updateStock(db, stockReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			fmt.Println("エラーだちょ")
			return
		}

		stock, err := getStock(db, stockReq.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stock)
	}
}

func updateStock(db Storer, stockReq Stock) error {
	_, err := db.Exec("INSERT INTO stocks (name, amount) VALUES (?, ?) ON DUPLICATE KEY UPDATE amount = amount + ?", stockReq.Name, stockReq.Amount, stockReq.Amount)
	fmt.Println("stockReq.Name : ", stockReq.Name)
	fmt.Println("stockReq.Amount : ", stockReq.Amount)
	fmt.Println(err)
	return err
}
func getStock(db Storer, name string) (Stock, error) {
	var stock Stock
	err := db.QueryRow("SELECT * FROM stocks WHERE name = ?", name).Scan(&stock.Name, &stock.Amount)
	return stock, err
}

// getAllStocks はすべての在庫を取得します。
func getAllStocks(db Storer) ([]Stock, error) {
	rows, err := db.Query("SELECT * FROM stocks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []Stock
	for rows.Next() {
		var stock Stock
		if err := rows.Scan(&stock.Name, &stock.Amount); err != nil {
			return nil, err
		}
		stocks = append(stocks, stock)
	}
	return stocks, nil
}
