package store

import (
	"context"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
)

// Client represents a PagerDuty API client wrapper
type Client struct {
	pd *pagerduty.Client
}

// NewClient creates a new PagerDuty client with the provided API key
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := pagerduty.NewClient(apiKey)
	return &Client{pd: client}, nil
}

// GetCurrentUser retrieves the current user from PagerDuty
func (c *Client) GetCurrentUser() (*pagerduty.User, error) {
	options := pagerduty.GetCurrentUserOptions{}
	user, err := c.pd.GetCurrentUserWithContext(context.TODO(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return user, nil
}