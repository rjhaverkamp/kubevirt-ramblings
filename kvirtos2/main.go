package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt/kvirtos2/internal/handlers"
	"github.com/kubevirt/kvirtos2/internal/kubevirt"
)

func main() {
	var (
		kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not provided)")
		namespace  = flag.String("namespace", "default", "Kubernetes namespace to use")
		port       = flag.String("port", "8080", "Port to listen on")
	)
	flag.Parse()

	// Create KubeVirt client
	kubevirtClient, err := kubevirt.NewClient(*kubeconfig, *namespace)
	if err != nil {
		log.Fatalf("Failed to create KubeVirt client: %v", err)
	}

	// Create handlers
	serverHandler := handlers.NewServerHandler(kubevirtClient)

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Nova API compatible endpoints
	v2 := router.Group("/v2.1")
	{
		// Server endpoints
		v2.POST("/servers", serverHandler.CreateServer)
		v2.GET("/servers", serverHandler.ListServers)
		v2.GET("/servers/detail", serverHandler.ListServers) // Alias for detailed listing
		v2.GET("/servers/:id", serverHandler.GetServer)
		v2.DELETE("/servers/:id", serverHandler.DeleteServer)
		v2.POST("/servers/:id/action", serverHandler.ServerAction)
	}

	// Also support the legacy /v2 endpoint for compatibility
	v2Legacy := router.Group("/v2")
	{
		v2Legacy.POST("/servers", serverHandler.CreateServer)
		v2Legacy.GET("/servers", serverHandler.ListServers)
		v2Legacy.GET("/servers/detail", serverHandler.ListServers)
		v2Legacy.GET("/servers/:id", serverHandler.GetServer)
		v2Legacy.DELETE("/servers/:id", serverHandler.DeleteServer)
		v2Legacy.POST("/servers/:id/action", serverHandler.ServerAction)
	}

	// Start server
	listenAddr := ":" + *port
	log.Printf("Starting kvirtos2 server on %s", listenAddr)
	log.Printf("Using namespace: %s", *namespace)
	if *kubeconfig != "" {
		log.Printf("Using kubeconfig: %s", *kubeconfig)
	} else {
		log.Printf("Using in-cluster configuration")
	}

	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
