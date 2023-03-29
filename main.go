package main

import (
	"blog/handlers"
	"context"
	"log"

	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	ctx := context.Background()

	// Test the connection to Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	router.HTMLRender = ginview.Default()
	router.StaticFile("/output.css", "./views/output.css")

	router.GET("/", func(c *gin.Context) {
		handlers.RenderPosts(c, client, ctx)
	})

	router.GET("/:id", func(c *gin.Context) {
		handlers.RenderPostByID(c, client, ctx)
	})

	router.GET("/posts", func(c *gin.Context) {
		handlers.GetPosts(c, client, ctx)
	})

	router.GET("/posts/:id", func(c *gin.Context) {
		handlers.GetPostByID(c, client, ctx)
	})

	router.GET("/build", func(c *gin.Context) {
		handlers.Build(c, client, ctx)
	})

	router.Run("localhost:8080")
}
