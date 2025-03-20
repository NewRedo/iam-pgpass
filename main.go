package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

const (
	OK int = iota
	ARG_ERR
	CREATE_PIPE_ERR
	OPEN_ERR
	ACCEPT_ERR
	AWS_ERR
)

const (
	DEFAULT_HOST     = "localhost"
	DEFAULT_PORT     = "5432"
	DEFAULT_USER     = "postgres"
	DEFAULT_DATABASE = "postgres"
)

type pgConn struct {
	host     string
	port     string
	user     string
	database string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "<pipe path>")
		os.Exit(ARG_ERR)
	}
	pipePath := os.Args[1]
	if err := checkPipe(pipePath); err != nil {
		fmt.Println("Error checking pipe:", err)
		os.Exit(CREATE_PIPE_ERR)
	}

	var host, port, user, database string
	if host = os.Getenv("PGHOST"); host == "" {
		host = DEFAULT_HOST
	}
	if port = os.Getenv("PGPORT"); port == "" {
		port = DEFAULT_PORT
	}
	if user = os.Getenv("PGUSER"); user == "" {
		user = DEFAULT_USER
	}
	if database = os.Getenv("PGDATABASE"); database == "" {
		database = DEFAULT_DATABASE
	}
	pg := &pgConn{host, port, user, database}

	os.Exit(run(pipePath, pg))
}

func checkPipe(pipePath string) error {
	if stat, err := os.Stat(pipePath); err == nil {
		if stat.Mode()&os.ModeNamedPipe == 0 {
			return fmt.Errorf("path exists but is not a pipe: %v", pipePath)
		}
		fmt.Println("Pipe already exists:", pipePath)
	} else {
		if err := syscall.Mkfifo(pipePath, 0600); err != nil {
			return fmt.Errorf("creating pipe: %w", err)
		}
		fmt.Println("Pipe created:", pipePath)
	}
	if err := os.Chmod(pipePath, 0600); err != nil {
		return fmt.Errorf("setting pipe permissions: %w", err)
	}
	return nil
}

func run(pipePath string, pg *pgConn) int {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	// Register for SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to communicate connection errors
	errChan := make(chan error)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		return AWS_ERR
	}

	// Handle connections in a separate goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				// If the context is canceled, exit the loop
				fmt.Println("Shutting down gracefully...")
				return
			default:
				fmt.Println("Waiting for client to connect...")

				// Open the pipe in read-only mode first (this will block until a writer opens the pipe)
				pipe, err := os.OpenFile(pipePath, os.O_WRONLY, os.ModeNamedPipe)
				if err != nil {
					if err == syscall.EINTR {
						// If interrupted (e.g., by a signal), just continue
						continue
					}
					errChan <- err
					return
				}

				fmt.Println("Client connected")

				// Generate and write credentials
				credentials, err := generateCredentials(ctx, pg, cfg)
				if err != nil {
					fmt.Println("Error generating credentials:", err)
					pipe.Close()
					continue // Don't exit the loop for credential generation errors
				}
				if _, err := pipe.Write([]byte(credentials)); err != nil {
					fmt.Println("Error writing to pipe:", err)
					pipe.Close()
					// Don't exit on write error, just wait for next client
					continue
				}

				pipe.Close()
				fmt.Println("Credentials sent, waiting for next client")
			}
		}
	}()

	// Wait for either a signal or an error
	select {
	case <-sigChan:
		fmt.Println("Interrupt received, canceling context...")
		cancel() // tells the goroutine to stop
		return OK
	case err := <-errChan:
		fmt.Println("Error accepting:", err)
		cancel() // also stop the goroutine
		return ACCEPT_ERR
	}
}

func generateCredentials(ctx context.Context, pg *pgConn, cfg aws.Config) (string, error) {
	token, err := auth.BuildAuthToken(ctx, fmt.Sprint(pg.host, ":", pg.port), cfg.Region, pg.user, cfg.Credentials)
	if err != nil {
		return "", fmt.Errorf("generating auth token: %w", err)
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s\n", pg.host, pg.port, pg.database, pg.user, token), nil
}
