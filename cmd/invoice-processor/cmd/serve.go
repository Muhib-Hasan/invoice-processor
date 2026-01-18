package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rezonia/invoice-processor/internal/server"
)

var (
	serverAddr     string
	serverDebug    bool
	readTimeout    time.Duration
	writeTimeout   time.Duration
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long: `Start an HTTP API server for processing invoices.

The API provides endpoints for:
  - POST /api/v1/process/xml    - Process XML invoice
  - POST /api/v1/process/pdf    - Process PDF invoice
  - POST /api/v1/process/image  - Process image invoice
  - POST /api/v1/process/auto   - Auto-detect and process
  - POST /api/v1/validate       - Validate invoice
  - POST /api/v1/info           - Get file information
  - GET  /health                - Health check

Examples:
  # Start server on default port
  invoice-processor serve

  # Start on custom port with API key
  invoice-processor serve --address :8080 --api-key <key>

  # Start in debug mode
  invoice-processor serve --debug`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&serverAddr, "address", ":8080", "Server listen address")
	serveCmd.Flags().BoolVar(&serverDebug, "debug", false, "Enable debug mode")
	serveCmd.Flags().DurationVar(&readTimeout, "read-timeout", 30*time.Second, "HTTP read timeout")
	serveCmd.Flags().DurationVar(&writeTimeout, "write-timeout", 5*time.Minute, "HTTP write timeout")
}

func runServe(cmd *cobra.Command, args []string) error {
	config := &server.Config{
		Address:        serverAddr,
		APIKey:         apiKey,
		LLMBaseURL:     llmBaseURL,
		LLMModel:       llmModel,
		LLMVisionModel: llmVisionModel,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		Debug:          serverDebug,
	}

	srv := server.NewServer(config)

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down server...")
		os.Exit(0)
	}()

	fmt.Printf("Starting server on %s\n", serverAddr)
	if apiKey != "" {
		fmt.Println("LLM extraction enabled")
	} else {
		fmt.Println("LLM extraction disabled (no API key)")
	}

	return srv.Run()
}
