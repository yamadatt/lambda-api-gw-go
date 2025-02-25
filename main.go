package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	fmt.Print(db) //debug
	r := gin.Default()
	setupRoutes(r, db) // ルーティング設定を routes.go に移動

	r.Run(":8080")
}
