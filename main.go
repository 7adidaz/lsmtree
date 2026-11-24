package main

import (
	"io"
	"log"
	"main/interfaces"
	"main/keys"
	"main/lsmtree"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
)

func main() {
	var threshold uint32 = 50
	var sparsityFactor uint32 = 2
	var falsePositiveRate float64 = 0.1

	println("Initializing LSM Tree with:")
	println("Threshold:", threshold)
	println("Sparsity factor:", sparsityFactor)
	println("False positive rate:", falsePositiveRate)

	go func() {
		log.Println("pprof listening on http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	r := gin.Default()

	dataPath, err := lsmtree.DataPath()
	if err != nil {
		log.Fatal(err)
	}
	lsm, err := lsmtree.NewLSMTree(dataPath, threshold, sparsityFactor, falsePositiveRate)
	if err != nil {
		log.Fatal(err)
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/:key", func(c *gin.Context) {
		key := c.Params.ByName("key")

		parsed_key := parseKey(key)
		found, value, err := lsm.Get(parsed_key)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get the key")
			return
		}
		if !found {
			c.String(http.StatusNotFound, "key is not found")
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
		if err := lsm.Delete(parsed_key); err != nil {
			c.String(http.StatusInternalServerError, "something went wrong deleting the key")
			return
		}

		c.String(http.StatusOK, "Key: "+key+" is deleted\n")
	})

	r.Run()
}

func parseKey(key string) interfaces.Comparable {
	var parsed_key interfaces.Comparable = keys.NewStringKey(key)
	return parsed_key
}
