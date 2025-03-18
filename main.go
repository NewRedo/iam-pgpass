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
	PERM_ERR
	ACCEPT_ERR
	AWS_ERR
)

const (
	DEFAULT_HOST     = "localhost"
	DEFAULT_PORT     = "5432"
	DEFAULT_USER     = "postgres"
	DEFAULT_DATABASE = "postgres"
)

type PgConn struct {
	host     string
	port     string
	user     string
	database string
}

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], " <pipe path>")
		return ARG_ERR
	}
	pipePath := os.Args[1]

	if stat, err := os.Stat(pipePath); err == nil {
		if stat.Mode()&os.ModeNamedPipe == 0 {
			fmt.Println("Path exists but is not a pipe:", pipePath)
			return CREATE_PIPE_ERR
		}
		fmt.Println("Pipe already exists:", pipePath)
	} else {
		if err := syscall.Mkfifo(pipePath, 0600); err != nil {
			fmt.Println("Error creating pipe:", err)
			return CREATE_PIPE_ERR
		}
	}

	fmt.Println("Listening on", pipePath)

	if err := os.Chmod(pipePath, 0600); err != nil {
		fmt.Println("Error setting pipe permissions:", err)
		return PERM_ERR
	}

	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	// Register for SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to communicate connection errors
	errChan := make(chan error)

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
	pg := &PgConn{host, port, user, database}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		return AWS_ERR
	}

	// Handle connections in a separate goroutine
	go func() {
		for {
			// Block until a reader opens the pipe
			pipe, err := os.OpenFile(pipePath, os.O_WRONLY, os.ModeNamedPipe)
			if err != nil {
				errChan <- err
				return
			}

			fmt.Println("Client connected")

			// Write credentials and close immediately
			credentials, err := generateCredentials(pg, cfg)
			if err != nil {
				fmt.Println("Error generating credentials:", err)
				pipe.Close()
				errChan <- err
				return
			}
			if _, err := pipe.Write([]byte(credentials)); err != nil {
				fmt.Println("Error writing to pipe:", err)
				pipe.Close()
				errChan <- err
				return
			}

			pipe.Close()
			fmt.Println("Credentials sent, waiting for next client")
		}
	}()

	// Wait for either a signal or an error
	select {
	case <-sigChan:
		fmt.Println("Interrupt received, shutting down...")
		return OK
	case err := <-errChan:
		fmt.Println("Error accepting:", err)
		return ACCEPT_ERR
	}
}

func generateCredentials(pg *PgConn, cfg aws.Config) (string, error) {
	token, err := auth.BuildAuthToken(context.TODO(), fmt.Sprint(pg.host, ":", pg.port), "eu-west-2", pg.user, cfg.Credentials)
	if err != nil {
		return "", fmt.Errorf("generating auth token: %w", err)
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s\n", pg.host, pg.port, pg.database, pg.user, token), nil
}
