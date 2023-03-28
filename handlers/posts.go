package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type Post struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Created string   `json:"created"`
	Tags    []string `json:"tags"`
}

const POST_PREFIX = "post-"
const POST_SET = "posts"
const LAYOUT = "2006-01-02"

func GetPosts(c *gin.Context, client *redis.Client, ctx context.Context) {
	// Retrieve the page number from the query parameters

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{
			"error": "can not parse page to int",
		})
		return
	}
	// Retrieve the list of post IDs from the zset in Redis
	start := (page - 1) * 20
	end := start + 19
	ids, err := client.ZRevRange(ctx, POST_SET, int64(start), int64(end)).Result()
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{
			"error": "Error retrieving post IDs",
		})
		return
	}

	// Retrieve the post data for each ID from Redis
	var response []Post
	for _, idStr := range ids {
		postJSON, err := client.Get(ctx, idStr).Result()
		if err != nil {
			log.Printf("Error retrieving post data for ID %s: %v", idStr, err)
			continue
		}
		var post Post
		err = json.Unmarshal([]byte(postJSON), &post)
		if err != nil {
			log.Printf("Error parsing post data for ID %s: %v", idStr, err)
			continue
		}
		response = append(response, post)
	}

	c.IndentedJSON(http.StatusOK, response)

}

func Build(c *gin.Context, client *redis.Client, ctx context.Context) {
	posts_dir := "./posts"
	files, err := ioutil.ReadDir(posts_dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		os.Exit(1)
	}
	var msg []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".md") {
			filePath := filepath.Join(posts_dir, file.Name())
			fmt.Println("Reading file:", filePath)

			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading file %s: %v\n", filePath, err)
				continue
			}

			// Split the file content into lines
			lines := strings.Split(string(content), "\n")

			// Parse the JSON metadata from the first line
			var metadata Post
			err = json.Unmarshal([]byte(lines[0]), &metadata)
			if err != nil {
				fmt.Printf("Error parsing metadata from file %s: %v\n", filePath, err)
				continue
			}
			metadata.Body = strings.Join(lines[1:], "\n")

			// Print the metadata
			fmt.Printf("File Name: %s, Title: %s\nCreated: %s\nTags: %v\n", filePath, metadata.Title, metadata.Created, metadata.Tags)
			msg = append(msg, fmt.Sprintf("Filename: %s, Title: %s, Created: %s, Tags: %v", filePath, metadata.Title, metadata.Created, metadata.Tags))

			metadataBytes, err := json.Marshal(metadata)
			if err != nil {
				fmt.Printf("Error marshaling metadata for file %s: %v\n", filePath, err)
				continue
			}
			fileName := strings.Replace(filePath, ".md", "", -1)
			postKey := POST_PREFIX + strings.Split(fileName, string(os.PathSeparator))[1]
			date, err := time.Parse(LAYOUT, metadata.Created)
			if err != nil {
				fmt.Println("Error parsing date:", err)
				return
			}

			timestamp := date.Unix()
			err = client.ZAdd(ctx, POST_SET, &redis.Z{
				Score:  float64(timestamp),
				Member: postKey,
			}).Err()

			if err != nil {
				fmt.Printf("Error saving %s to Redis zset %s: %v\n", postKey, POST_SET, err)
				continue
			}

			err = client.Set(ctx, postKey, metadataBytes, 0).Err()

			if err != nil {
				fmt.Printf("Error saving post to Redis for file %s: %v\n", filePath, err)
				continue
			}

		}
	}

	c.IndentedJSON(http.StatusOK, msg)
}

func GetPostByID(c *gin.Context, client *redis.Client, ctx context.Context) {
	postID := c.Param("id")
	postRaw, err := client.Get(ctx, POST_PREFIX+postID).Result()

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Can not find %s ", postID),
		})
		return
	}
	var post Post
	json.Unmarshal([]byte(postRaw), &post)

	c.IndentedJSON(http.StatusOK, post)
}
