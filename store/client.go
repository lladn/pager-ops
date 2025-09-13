package store

import (
	"context"
	"fmt"
	"pager-ops/database"
	"time"

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

// FetchOptions provides flexible options for fetching incidents
type FetchOptions struct {
	ServiceIDs []string
	Statuses   []string
	Since      time.Time
	Until      time.Time
	UserID     string
}

// FetchIncidentsWithOptions fetches incidents with flexible filtering options
func (c *Client) FetchIncidentsWithOptions(opts FetchOptions) ([]database.IncidentData, error) {
	pdOpts := pagerduty.ListIncidentsOptions{
		Statuses:   opts.Statuses,
		ServiceIDs: opts.ServiceIDs,
		Limit:      100,
		SortBy:     "created_at:desc",
	}

	// Add time filters if provided
	if !opts.Since.IsZero() {
		pdOpts.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		pdOpts.Until = opts.Until.Format(time.RFC3339)
	}

	// Add user filter if provided
	if opts.UserID != "" {
		pdOpts.UserIDs = []string{opts.UserID}
	}

	var allIncidents []database.IncidentData
	offset := uint(0)

	for {
		pdOpts.Offset = offset
		resp, err := c.pd.ListIncidentsWithContext(context.TODO(), pdOpts)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More {
			break
		}
		offset += pdOpts.Limit
	}

	return allIncidents, nil
}
