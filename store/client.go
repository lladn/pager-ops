package store

import (
	"context"
	"fmt"
	"pager-ops/database"
	"sync"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// Client represents a PagerDuty API client wrapper - STRUCT ENHANCED WITH NEW FIELDS
type Client struct {
	pd            *pagerduty.Client
	// New fields for enhanced tracking
	mu            sync.RWMutex
	lastCallTime  time.Time
	callCount     int
}

// NewClient creates a new PagerDuty client - ORIGINAL METHOD UNCHANGED
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := pagerduty.NewClient(apiKey)
	return &Client{
		pd: client,
	}, nil
}

// GetCurrentUser retrieves the current user - ORIGINAL METHOD ENHANCED
func (c *Client) GetCurrentUser() (*pagerduty.User, error) {
	c.recordAPICall()
	
	options := pagerduty.GetCurrentUserOptions{}
	user, err := c.pd.GetCurrentUserWithContext(context.TODO(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return user, nil
}

// FetchOptions provides flexible options - NEW STRUCT FOR ENHANCED FUNCTIONALITY
type FetchOptions struct {
	ServiceIDs []string
	Statuses   []string
	Since      time.Time
	Until      time.Time
	UserID     string
	Limit      uint
}

// NEW METHOD - FetchIncidentsWithOptions for flexible incident fetching
// This is used by app.go for fetching resolved incidents with time windows
func (c *Client) FetchIncidentsWithOptions(opts FetchOptions) ([]database.IncidentData, error) {
	c.recordAPICall()
	
	pdOpts := pagerduty.ListIncidentsOptions{
		Statuses:   opts.Statuses,
		ServiceIDs: opts.ServiceIDs,
		Limit:      100,
		SortBy:     "created_at:desc",
	}

	if opts.Limit > 0 {
		pdOpts.Limit = opts.Limit
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
			return nil, fmt.Errorf("failed to list incidents: %w", err)
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More {
			break
		}
		offset += pdOpts.Limit
		
		// Record additional API calls for pagination
		c.recordAPICall()
	}

	return allIncidents, nil
}

// NEW METHOD - FetchIncidentsWithPagination for controlled pagination
// This is used by app.go for the initial 72-hour fetch with pagination control
func (c *Client) FetchIncidentsWithPagination(opts FetchOptions, pageSize uint) ([]database.IncidentData, error) {
	c.recordAPICall()
	
	if pageSize == 0 {
		pageSize = 100
	}

	pdOpts := pagerduty.ListIncidentsOptions{
		Statuses:   opts.Statuses,
		ServiceIDs: opts.ServiceIDs,
		Limit:      pageSize,
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
	maxPages := 10 // Limit to prevent runaway pagination

	for page := 0; page < maxPages; page++ {
		pdOpts.Offset = offset
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		resp, err := c.pd.ListIncidentsWithContext(ctx, pdOpts)
		cancel()
		
		if err != nil {
			return allIncidents, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More {
			break
		}
		
		offset += pageSize
		c.recordAPICall()
		
		// Small delay between pages to be respectful of rate limits
		time.Sleep(100 * time.Millisecond)
	}

	return allIncidents, nil
}

// NEW PRIVATE METHOD - recordAPICall for tracking API usage
func (c *Client) recordAPICall() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	if now.Sub(c.lastCallTime) > time.Minute {
		// Reset counter if more than a minute has passed
		c.callCount = 1
		c.lastCallTime = now
	} else {
		c.callCount++
	}
}

// NEW METHOD - GetAPICallStats returns current API call statistics
func (c *Client) GetAPICallStats() (int, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.callCount, c.lastCallTime
}