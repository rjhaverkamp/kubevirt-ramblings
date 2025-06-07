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

	// Create v1 handler
	v1Handler := handlers.NewV1Handler(kubevirtClient)

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
	router.GET("/health", v1Handler.Health)

	// V1 API endpoints
	v1 := router.Group("/api/v1")
	{
		// VM endpoints
		v1.GET("/vms", v1Handler.ListVMs)
		v1.POST("/vms", v1Handler.CreateVM)
		v1.GET("/vms/:name", v1Handler.GetVM)
		v1.PUT("/vms/:name", v1Handler.UpdateVM)
		v1.DELETE("/vms/:name", v1Handler.DeleteVM)
		
		// VM actions
		v1.POST("/vms/:name/start", v1Handler.StartVM)
		v1.POST("/vms/:name/stop", v1Handler.StopVM)
		v1.POST("/vms/:name/restart", v1Handler.RestartVM)
		v1.GET("/vms/:name/console", v1Handler.GetVMConsole)
	}

	// Start server
	listenAddr := ":" + *port
	log.Printf("Starting kvirtos2 server on %s", listenAddr)
	log.Printf("Using namespace: %s", *namespace)
	log.Printf("V1 API available at: /api/v1/vms")
	if *kubeconfig != "" {
		log.Printf("Using kubeconfig: %s", *kubeconfig)
	} else {
		log.Printf("Using in-cluster configuration")
	}

	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
