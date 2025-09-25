package main

import (
	"io"
	"main/interfaces"
	"main/keys"
	"main/lsmtree"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	lsm := lsmtree.NewLSMTree(2, 10, 0.01)

	// Define a simple GET endpoint
	r.GET("/ping", func(c *gin.Context) {
		// Return JSON response
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/:key", func(c *gin.Context) {
		key := c.Params.ByName("key")

		parsed_key := parseKey(key)
		found, value, err := lsm.Get(parsed_key)
		if !found {
			c.String(http.StatusNotFound, "key is not found")
			return
		}
		if err != nil {
			c.String(http.StatusTeapot, "Failed to get the key")
			return
		}
		c.String(http.StatusOK, string(value))
	})

	r.PUT("/:key", func(c *gin.Context) {
		key := c.Params.ByName("key")

		defer c.Request.Body.Close()
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusTeapot, "Failed to read request body")
			return
		}

		parsed_key := parseKey(key)

		err = lsm.Put(parsed_key, body)
		if err != nil {
			c.String(http.StatusInternalServerError, "something went wrong putting the key")
			return
		}

		c.String(http.StatusOK, "Key: "+key+" is set\n")
	})

	r.DELETE("/:key", func(c *gin.Context) {
		key := c.Params.ByName("key")

		parsed_key := parseKey(key)
		lsm.Delete(parsed_key)

		c.String(http.StatusOK, "Key: "+key+" is deleted\n")
	})

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}


func parseKey(key string) interfaces.Comparable {
	var parsed_key interfaces.Comparable = keys.NewStringKey(key)
	// num, err := strconv.Atoi(key)
	// if err != nil {
	// 	parsed_key = keys.NewIntKey(uint32(num))
	// }

	return parsed_key
}
