// Package main is the entry point for the GitHub MCP Server.
// It initializes and starts the Model Context Protocol server that exposes
// GitHub API functionality as MCP tools.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/github/github-mcp-server/pkg/server"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags.
	Version = "dev"
	// Commit is the git commit hash set at build time.
	Commit = "unknown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var (
		token   string
		logFile string
		readOnly bool
	)

	cmd := &cobra.Command{
		Use:   "github-mcp-server",
		Short: "GitHub MCP Server - exposes GitHub API as MCP tools",
		Long: `GitHub MCP Server implements the Model Context Protocol (MCP)
and exposes GitHub API functionality as tools that can be used by
AI assistants and other MCP clients.`,
		Version: fmt.Sprintf("%s (%s)", Version, Commit),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), token, logFile, readOnly)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "GitHub personal access token (or set GITHUB_TOKEN env var)")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Path to log file (default: stderr)")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Restrict server to read-only operations only")

	return cmd
}

func runServer(ctx context.Context, token, logFile string, readOnly bool) error {
	// Resolve token from flag or environment variable.
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("GitHub token is required: set --token flag or GITHUB_TOKEN environment variable")
	}

	// Set up context that cancels on OS interrupt signals.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Build and start the MCP server.
	srv, err := server.New(server.Options{
		Token:    token,
		LogFile:  logFile,
		ReadOnly: readOnly,
		Version:  Version,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "GitHub MCP Server %s starting (stdio transport)\n", Version)

	if err := srv.ServeStdio(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
