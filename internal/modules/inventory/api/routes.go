package inventoryApi

import (
	"crypto/sha256"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func InventoryLoadHandler(c *gin.Context) {
	var wg sync.WaitGroup
	numRoutines := 2500

	// Create a coordination channel to hold the goroutines open
	holdChan := make(chan struct{})

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Perform CPU work
			data := []byte("inventory-sustained-cpu-load")
			_ = sha256.Sum256(data)

			// HOLD: Stay alive until the channel is closed.
			// This forces all 2,500 to remain active concurrently.
			<-holdChan
		}(i)
	}

	// Wait 5 seconds to give Prometheus plenty of time to scrape the 2,500 concurrent routines
	time.Sleep(5 * time.Second)

	// Release all goroutines at once
	close(holdChan)
	wg.Wait()

	c.JSON(200, gin.H{
		"service": "inventoryApi",
		"profile": "Sustained Goroutine Explosion",
		"status":  "success",
		"active":  numRoutines,
	})
}

func SetupRoutes(r *gin.Engine, h *ApiHandler) {
	baseGroup := r.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)

	api := r.Group("/test")
	{
		api.GET("/load", InventoryLoadHandler)
	}
}
