package main

import (
	"github.com/gin-gonic/gin"
)

// setupRoutes は Gin のルーティングを設定します。
func setupRoutes(r *gin.Engine, db Storer) {
	v1 := r.Group("/v1")
	{
		v1.GET("/stocks/:name", getStocksHandler(db))
		v1.GET("/stocks", getAllStocksHandler(db))
		v1.POST("/stocks", postStocksHandler(db))
	}
}
