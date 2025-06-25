package main

import (
	"log"
	"net/http"

	"gwapi-graph/internal/api"
	"gwapi-graph/internal/k8s"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create API handler
	apiHandler := api.NewHandler(k8sClient)

	// Setup Gin router
	r := gin.Default()

	// Serve static files
	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("web/templates/*")

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Gateway API Graph Visualizer",
		})
	})

	// API routes
	api := r.Group("/api")
	{
		api.GET("/resources", apiHandler.GetResources)
		api.GET("/graph", apiHandler.GetGraph)
		api.GET("/ws", apiHandler.HandleWebSocket)
		api.GET("/resource/:type/:name", apiHandler.GetResourceDetails)
		api.PUT("/resource/:type/:name", apiHandler.UpdateResource)
	}

	log.Println("Starting server on :8080")
	r.Run(":8080")
}
