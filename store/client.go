package store

import (
	"context"
	"fmt"
	"pager-ops/database"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// APIRequest represents a queued API request
type APIRequest struct {
	Type       string
	Context    context.Context
	Options    interface{}
	ResultChan chan APIResponse
}

// APIResponse represents the response from an API call
type APIResponse struct {
	Data  interface{}
	Error error
}

// APIQueue manages rate-limited API calls
type APIQueue struct {
	requestChan chan *APIRequest
	stopChan    chan struct{}
	wg          sync.WaitGroup

	// Rate limiting
	maxCallsPerMinute int
	callTimes         []time.Time
	mu                sync.Mutex

	// Metrics
	totalCalls  int64
	failedCalls int64
	metricsmu   sync.RWMutex
}

// Client represents a PagerDuty API client wrapper with queue
type Client struct {
	pd       *pagerduty.Client
	apiQueue *APIQueue
	logger   func(string)
}

// NewClient creates a new PagerDuty client with API queue
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	pdClient := pagerduty.NewClient(apiKey)

	// Initialize API queue
	queue := &APIQueue{
		requestChan:       make(chan *APIRequest, 100), // Buffer for 100 requests
		stopChan:          make(chan struct{}),
		maxCallsPerMinute: 600, // Conservative: 600 calls/min (PagerDuty allows 960)
		callTimes:         make([]time.Time, 0),
	}

	client := &Client{
		pd:       pdClient,
		apiQueue: queue,
		logger:   func(msg string) { fmt.Println(msg) }, // Default logger
	}

	// Start the API queue worker
	queue.wg.Add(1)
	go client.processAPIQueue()

	return client, nil
}

// SetLogger allows setting a custom logger
func (c *Client) SetLogger(logger func(string)) {
	c.logger = logger
}

// Shutdown gracefully stops the API queue
func (c *Client) Shutdown() {
	close(c.apiQueue.stopChan)
	c.apiQueue.wg.Wait()
	close(c.apiQueue.requestChan)
}

// processAPIQueue is the main worker that processes API requests
func (c *Client) processAPIQueue() {
	defer c.apiQueue.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-c.apiQueue.stopChan:
			// Process remaining requests before shutdown
			for len(c.apiQueue.requestChan) > 0 {
				req := <-c.apiQueue.requestChan
				c.executeAPICall(req)
			}
			return

		case req := <-c.apiQueue.requestChan:
			// Wait if rate limit would be exceeded
			c.waitForRateLimit()
			c.executeAPICall(req)

		case <-ticker.C:
			// Periodic cleanup of old call times
			c.cleanupCallTimes()
		}
	}
}

// waitForRateLimit ensures we don't exceed rate limits
func (c *Client) waitForRateLimit() {
	c.apiQueue.mu.Lock()
	defer c.apiQueue.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	// Count calls in the last minute
	validCalls := []time.Time{}
	for _, callTime := range c.apiQueue.callTimes {
		if callTime.After(windowStart) {
			validCalls = append(validCalls, callTime)
		}
	}
	c.apiQueue.callTimes = validCalls

	// If at limit, calculate wait time
	if len(validCalls) >= c.apiQueue.maxCallsPerMinute {
		oldestCall := validCalls[0]
		waitDuration := oldestCall.Add(1 * time.Minute).Sub(now)
		if waitDuration > 0 {
			c.logger(fmt.Sprintf("Rate limit reached, waiting %v", waitDuration))
			time.Sleep(waitDuration)
		}
	}

	// Add small delay between calls to smooth out bursts
	if len(validCalls) > 0 {
		time.Sleep(100 * time.Millisecond)
	}

	// Record this call
	c.apiQueue.callTimes = append(c.apiQueue.callTimes, now)
}

// cleanupCallTimes removes old entries from call tracking
func (c *Client) cleanupCallTimes() {
	c.apiQueue.mu.Lock()
	defer c.apiQueue.mu.Unlock()

	windowStart := time.Now().Add(-1 * time.Minute)
	validCalls := []time.Time{}
	for _, callTime := range c.apiQueue.callTimes {
		if callTime.After(windowStart) {
			validCalls = append(validCalls, callTime)
		}
	}
	c.apiQueue.callTimes = validCalls
}

// executeAPICall performs the actual API call based on request type
func (c *Client) executeAPICall(req *APIRequest) {
	atomic.AddInt64(&c.apiQueue.totalCalls, 1)

	var result interface{}
	var err error

	// Process based on request type
	switch req.Type {
	case "GetCurrentUser":
		opts := req.Options.(pagerduty.GetCurrentUserOptions)
		result, err = c.pd.GetCurrentUserWithContext(req.Context, opts)

	case "ListIncidents":
		opts := req.Options.(pagerduty.ListIncidentsOptions)
		result, err = c.pd.ListIncidentsWithContext(req.Context, opts)

	case "ListIncidentAlerts":
		incidentID := req.Options.(string)
		result, err = c.pd.ListIncidentAlertsWithContext(req.Context, incidentID, pagerduty.ListIncidentAlertsOptions{})

	case "ListIncidentNotes":
		incidentID := req.Options.(string)
		result, err = c.pd.ListIncidentNotesWithContext(req.Context, incidentID)

	default:
		err = fmt.Errorf("unknown API request type: %s", req.Type)
	}

	if err != nil {
		// Increment failure counter atomically
		atomic.AddInt64(&c.apiQueue.failedCalls, 1)
		c.logger(fmt.Sprintf("API call failed: %s - %v", req.Type, err))
	}

	// Send response
	select {
	case req.ResultChan <- APIResponse{Data: result, Error: err}:
	case <-time.After(5 * time.Second):
		c.logger("Timeout sending API response")
	}
}

// queueRequest adds a request to the queue and waits for response
func (c *Client) queueRequest(reqType string, ctx context.Context, options interface{}) (interface{}, error) {
	req := &APIRequest{
		Type:       reqType,
		Context:    ctx,
		Options:    options,
		ResultChan: make(chan APIResponse, 1),
	}

	// Send request to queue with longer timeout
	select {
	case c.apiQueue.requestChan <- req:
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while queueing %s request", reqType)
	case <-time.After(30 * time.Second):
		// Log queue stats for debugging - USE ALL VARIABLES
		total, failed, pending := c.GetAPIStats()
		c.logger(fmt.Sprintf("Queue timeout: type=%s, pending=%d, total=%d, failed=%d",
			reqType, pending, total, failed))
		return nil, fmt.Errorf("timeout queueing %s request (queue may be full)", reqType)
	}

	// Wait for response with extended timeout for resolved incidents
	timeout := 60 * time.Second
	if strings.Contains(reqType, "ListIncidents") {
		// Check if it's a resolved incidents fetch (has "resolved" in statuses)
		if opts, ok := options.(pagerduty.ListIncidentsOptions); ok {
			for _, status := range opts.Statuses {
				if status == "resolved" {
					timeout = 90 * time.Second // Longer timeout for resolved
					break
				}
			}
		}
	}

	select {
	case resp := <-req.ResultChan:
		return resp.Data, resp.Error
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled waiting for %s response", reqType)
	case <-time.After(timeout):
		total, failed, pending := c.GetAPIStats()
		c.logger(fmt.Sprintf("Response timeout: type=%s, timeout=%v, pending=%d, total=%d, failed=%d",
			reqType, timeout, pending, total, failed))
		return nil, fmt.Errorf("timeout waiting for %s API response after %v", reqType, timeout)
	}
}

// GetCurrentUser retrieves the current user through the queue
func (c *Client) GetCurrentUser() (*pagerduty.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	options := pagerduty.GetCurrentUserOptions{}
	result, err := c.queueRequest("GetCurrentUser", ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	user, ok := result.(*pagerduty.User)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	return user, nil
}

// FetchOptions provides flexible options
type FetchOptions struct {
	ServiceIDs []string
	Statuses   []string
	Since      time.Time
	Until      time.Time
	UserID     string
	Limit      uint
}

// FetchOpenIncidents fetches open incidents with rate limiting
func (c *Client) FetchOpenIncidents(serviceIDs []string, userID string) ([]database.IncidentData, error) {
	var allIncidents []database.IncidentData

	// Fetch incidents filtered by services
	if len(serviceIDs) > 0 {
		serviceIncidents, err := c.fetchIncidentsByServices(
			serviceIDs, []string{"triggered", "acknowledged"})
		if err != nil {
			return nil, err
		}
		allIncidents = append(allIncidents, serviceIncidents...)
	}

	// Fetch incidents assigned to current user
	if userID != "" {
		userIncidents, err := c.fetchIncidentsByUser(
			userID, []string{"triggered", "acknowledged"})
		if err != nil {
			return nil, err
		}
		allIncidents = append(allIncidents, userIncidents...)
	}

	// Deduplicate incidents
	return deduplicateIncidents(allIncidents), nil
}

// fetchIncidentsByServices fetches incidents by service IDs through queue
func (c *Client) fetchIncidentsByServices(serviceIDs []string, statuses []string) ([]database.IncidentData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := pagerduty.ListIncidentsOptions{
		Statuses:   statuses,
		ServiceIDs: serviceIDs,
		Limit:      50,
		SortBy:     "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)
	maxPages := 2 // Limit to 100 incidents total

	for page := 0; page < maxPages; page++ {
		opts.Offset = offset

		result, err := c.queueRequest("ListIncidents", ctx, opts)
		if err != nil {
			return allIncidents, err // Return what we have
		}

		resp, ok := result.(*pagerduty.ListIncidentsResponse)
		if !ok {
			return allIncidents, fmt.Errorf("unexpected response type")
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More || len(allIncidents) >= 100 {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

// fetchIncidentsByUser fetches incidents by user ID through queue
func (c *Client) fetchIncidentsByUser(userID string, statuses []string) ([]database.IncidentData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := pagerduty.ListIncidentsOptions{
		Statuses: statuses,
		UserIDs:  []string{userID},
		Limit:    50,
		SortBy:   "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)
	maxPages := 2 // Limit to 100 incidents total

	for page := 0; page < maxPages; page++ {
		opts.Offset = offset

		result, err := c.queueRequest("ListIncidents", ctx, opts)
		if err != nil {
			return allIncidents, err // Return what we have
		}

		resp, ok := result.(*pagerduty.ListIncidentsResponse)
		if !ok {
			return allIncidents, fmt.Errorf("unexpected response type")
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More || len(allIncidents) >= 100 {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

// FetchResolvedIncidents fetches resolved incidents through queue
func (c *Client) FetchResolvedIncidents(serviceIDs []string) ([]database.IncidentData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	until := time.Now()
	since := until.AddDate(0, 0, -3)

	opts := pagerduty.ListIncidentsOptions{
		Statuses:   []string{"resolved"},
		ServiceIDs: serviceIDs,
		Since:      since.Format(time.RFC3339),
		Until:      until.Format(time.RFC3339),
		Limit:      50,
		SortBy:     "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)
	maxPages := 2

	for page := 0; page < maxPages; page++ {
		opts.Offset = offset

		result, err := c.queueRequest("ListIncidents", ctx, opts)
		if err != nil {
			return allIncidents, err
		}

		resp, ok := result.(*pagerduty.ListIncidentsResponse)
		if !ok {
			return allIncidents, fmt.Errorf("unexpected response type")
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More || len(allIncidents) >= 100 {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

// FetchIncidentsWithPagination for controlled pagination through queue
func (c *Client) FetchIncidentsWithPagination(opts FetchOptions, pageSize uint) ([]database.IncidentData, error) {
	timeout := 60 * time.Second
	for _, status := range opts.Statuses {
		if status == "resolved" {
			timeout = 120 * time.Second
			break
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if pageSize == 0 {
		pageSize = 50
	}

	pdOpts := pagerduty.ListIncidentsOptions{
		Statuses:   opts.Statuses,
		ServiceIDs: opts.ServiceIDs,
		Limit:      pageSize,
		SortBy:     "created_at:desc",
	}

	if !opts.Since.IsZero() {
		pdOpts.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		pdOpts.Until = opts.Until.Format(time.RFC3339)
	}
	if opts.UserID != "" {
		pdOpts.UserIDs = []string{opts.UserID}
	}

	var allIncidents []database.IncidentData
	offset := uint(0)
	maxPages := 2

	for page := 0; page < maxPages; page++ {
		pdOpts.Offset = offset

		result, err := c.queueRequest("ListIncidents", ctx, pdOpts)
		if err != nil {
			return allIncidents, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		resp, ok := result.(*pagerduty.ListIncidentsResponse)
		if !ok {
			return allIncidents, fmt.Errorf("unexpected response type")
		}

		for _, i := range resp.Incidents {
			incident := convertToIncidentData(i)
			allIncidents = append(allIncidents, incident)
		}

		if !resp.More || len(allIncidents) >= 100 {
			break
		}
		offset += pageSize
	}

	return allIncidents, nil
}

// FetchIncidentsWithOptions for flexible incident fetching through queue
func (c *Client) FetchIncidentsWithOptions(opts FetchOptions) ([]database.IncidentData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pdOpts := pagerduty.ListIncidentsOptions{
		Statuses:   opts.Statuses,
		ServiceIDs: opts.ServiceIDs,
		Limit:      100,
		SortBy:     "created_at:desc",
	}

	if opts.Limit > 0 {
		pdOpts.Limit = opts.Limit
	}
	if !opts.Since.IsZero() {
		pdOpts.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		pdOpts.Until = opts.Until.Format(time.RFC3339)
	}
	if opts.UserID != "" {
		pdOpts.UserIDs = []string{opts.UserID}
	}

	var allIncidents []database.IncidentData
	offset := uint(0)

	for {
		pdOpts.Offset = offset

		result, err := c.queueRequest("ListIncidents", ctx, pdOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list incidents: %w", err)
		}

		resp, ok := result.(*pagerduty.ListIncidentsResponse)
		if !ok {
			return nil, fmt.Errorf("unexpected response type")
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

// GetIncidentAlerts fetches alerts for a specific incident through queue
func (c *Client) GetIncidentAlerts(incidentID string) ([]IncidentAlert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.queueRequest("ListIncidentAlerts", ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incident alerts: %w", err)
	}

	resp, ok := result.(*pagerduty.ListAlertsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for alerts")
	}

	var alerts []IncidentAlert
	for _, alert := range resp.Alerts {
		convertedAlert := IncidentAlert{
			ID:        alert.ID,
			Summary:   alert.Summary,
			Status:    alert.Status,
			CreatedAt: alert.CreatedAt,
		}

		// Check if service exists (APIObject has ID field)
		if alert.Service.ID != "" {
			convertedAlert.ServiceName = alert.Service.Summary
		}

		// Extract context links from Body map if it exists
		if alert.Body != nil {
			// Try to extract CEF details if they exist
			if cefDetails, ok := alert.Body["cef_details"]; ok {
				if cefMap, ok := cefDetails.(map[string]interface{}); ok {
					// Try to extract contexts
					if contexts, ok := cefMap["contexts"]; ok {
						if contextList, ok := contexts.([]interface{}); ok {
							for _, ctx := range contextList {
								if contextMap, ok := ctx.(map[string]interface{}); ok {
									link := AlertLink{
										Href: getString(contextMap, "href"),
										Text: getString(contextMap, "text"),
									}
									if link.Href != "" {
										convertedAlert.Links = append(convertedAlert.Links, link)
									}
								}
							}
						}
					}
				}
			}
		}

		alerts = append(alerts, convertedAlert)
	}

	return alerts, nil
}

// GetIncidentNotes fetches notes for a specific incident through queue
func (c *Client) GetIncidentNotes(incidentID string) ([]IncidentNote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.queueRequest("ListIncidentNotes", ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incident notes: %w", err)
	}

	// The response is a slice of IncidentNote values (not pointers)
	resp, ok := result.([]pagerduty.IncidentNote)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for notes")
	}

	var notes []IncidentNote
	for _, note := range resp {
		convertedNote := IncidentNote{
			ID:        note.ID,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
		}

		// Check if User exists (APIObject has ID field)
		if note.User.ID != "" {
			convertedNote.UserName = note.User.Summary
		}

		notes = append(notes, convertedNote)
	}

	return notes, nil
}

// Helper function to safely get string from interface
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetAPIStats returns current API queue statistics
func (c *Client) GetAPIStats() (totalCalls int64, failedCalls int64, pendingRequests int) {
	c.apiQueue.metricsmu.RLock()
	defer c.apiQueue.metricsmu.RUnlock()

	return atomic.LoadInt64(&c.apiQueue.totalCalls),
		atomic.LoadInt64(&c.apiQueue.failedCalls),
		len(c.apiQueue.requestChan)
}
