// Package github provides utilities for interacting with the GitHub API
// and building MCP tools that wrap GitHub functionality.
package github

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with additional configuration.
type Client struct {
	*github.Client
	token string
}

// ClientOption is a functional option for configuring a GitHub client.
type ClientOption func(*Client)

// WithToken sets the authentication token for the GitHub client.
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// NewClient creates a new GitHub API client with the provided options.
// If no token is provided via options, it falls back to the GITHUB_TOKEN
// environment variable. For GitHub Enterprise, set GITHUB_API_URL or
// GITHUB_ENTERPRISE_URL to the base URL of your GitHub Enterprise instance.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{}

	for _, opt := range opts {
		opt(c)
	}

	// Fall back to environment variable if no token was set
	if c.token == "" {
		c.token = os.Getenv("GITHUB_TOKEN")
	}

	var httpClient *http.Client
	if c.token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	ghClient := github.NewClient(httpClient)

	// Support GitHub Enterprise via GITHUB_API_URL env var.
	// Also accept GITHUB_ENTERPRISE_URL as a convenient alias.
	apiURL := os.Getenv("GITHUB_API_URL")
	if apiURL == "" {
		apiURL = os.Getenv("GITHUB_ENTERPRISE_URL")
	}

	if apiURL != "" {
		var err error
		ghClient, err = ghClient.WithAuthToken(c.token).WithEnterpriseURLs(apiURL, apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure enterprise GitHub client: %w", err)
		}
	} else if c.token != "" {
		ghClient = ghClient.WithAuthToken(c.token)
	}

	c.Client = ghClient
	return c, nil
}

// IsAuthenticated returns true if the client has an authentication token configured.
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// GetAuthenticatedUser returns the currently authenticated GitHub user.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return user, nil
}

// RequireAuth returns an error if the client is not authenticated.
// This is useful for tools that require authentication to function.
// Note: unauthenticated requests are subject to a much lower rate limit
// (60 requests/hour vs 5000 requests/hour for authenticated requests).
func (c *Client) RequireAuth() error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required: set GITHUB_TOKEN environment variable or provide a token")
	}
	return nil
}
