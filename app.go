package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pager-ops/database"
	"pager-ops/store"

	"github.com/99designs/keyring"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx                context.Context
	db                 *database.DB
	client             *store.Client
	polling            bool
	pollTicker         *time.Ticker
	servicesConfig     *store.ServicesConfig
	selectedServices   []string
	kr                 keyring.Keyring
	logger             *Logger
	filterByUser       bool
	mu                 sync.RWMutex
	pollMu             sync.RWMutex
	notificationMgr    *NotificationManager
	lastIncidents      map[string]string
	lastIncidentsMu    sync.RWMutex
	resolvedPollTicker *time.Ticker
	resolvedPolling    bool
	resolvedPollMu     sync.RWMutex
	rateLimitTracker   *RateLimitTracker
}

type RateLimitTracker struct {
	mu         sync.Mutex
	calls      []time.Time
	windowSize time.Duration
	maxCalls   int
}

func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		calls:      make([]time.Time, 0),
		windowSize: time.Minute,
		maxCalls:   960, // 960 requests per minute per user
	}
}

// CanMakeCall checks if we can make an API call without exceeding rate limit
func (r *RateLimitTracker) CanMakeCall() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	// Remove old calls outside the window
	validCalls := make([]time.Time, 0)
	for _, callTime := range r.calls {
		if callTime.After(cutoff) {
			validCalls = append(validCalls, callTime)
		}
	}
	r.calls = validCalls

	return len(r.calls) < r.maxCalls
}

// RecordCall records an API call
func (r *RateLimitTracker) RecordCall() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, time.Now())
}

// GetCurrentRate returns the current call rate per minute
func (r *RateLimitTracker) GetCurrentRate() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	count := 0
	for _, callTime := range r.calls {
		if callTime.After(cutoff) {
			count++
		}
	}

	return count
}

// Set default filterByUser to true:
func NewApp() *App {
	return &App{
		filterByUser:  true,
		lastIncidents: make(map[string]string),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize logger
	logger, err := NewLogger()
	if err != nil {
		runtime.LogError(ctx, fmt.Sprintf("Failed to initialize logger: %v", err))
	} else {
		a.logger = logger
		a.logger.Info("PagerOps starting up...")
	}

	// Initialize database
	dbPath := filepath.Join(".", "incidents.db")
	a.logger.Info(fmt.Sprintf("Initializing database at: %s", dbPath))

	db, err := database.NewDB(dbPath)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to initialize database: %v", err))
		runtime.LogError(ctx, fmt.Sprintf("Failed to initialize database: %v", err))
		return
	}
	a.db = db
	a.logger.Info("Database initialized successfully")

	// Clear old incidents from database on startup to ensure fresh data
	if err := a.db.ClearIncidents(); err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to clear old incidents: %v", err))
	}

	// Initialize keyring
	kr, err := keyring.Open(keyring.Config{
		ServiceName: "PagerOps",
	})
	if err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to initialize keyring: %v", err))
		runtime.LogWarning(ctx, fmt.Sprintf("Failed to initialize keyring: %v", err))
	} else {
		a.kr = kr
		a.logger.Info("Keyring initialized successfully")
	}

	a.notificationMgr = NewNotificationManager(a.logger)
	a.logger.Info("Notification manager initialized")

	// Try to load API key and initialize client
	apiKey, err := a.GetAPIKey()
	if err == nil && apiKey != "" {
		client, err := store.NewClient(apiKey)
		if err == nil {
			a.client = client
			a.logger.Info("PagerDuty client initialized successfully")

			// Set default filter to true (show only assigned)
			a.filterByUser = true

			// Start polling immediately with fresh data
			a.StartPolling()
		} else {
			a.logger.Warn(fmt.Sprintf("Failed to initialize PagerDuty client: %v", err))
		}
	}

	a.rateLimitTracker = NewRateLimitTracker()

	// Start both polling mechanisms if client is initialized
	if a.client != nil {
		a.StartPolling()         // Existing 3-second polling for open incidents
		a.StartResolvedPolling() // New 10-second polling for resolved incidents
	}
}

// SetFilterByUser toggles filtering incidents by current user
func (a *App) SetFilterByUser(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filterByUser = enabled
	a.logger.Info(fmt.Sprintf("Filter by user set to: %v", enabled))

	// Trigger immediate update
	if a.polling {
		go a.fetchAndUpdateIncidents()
	}
}

// GetFilterByUser returns the current filter state
func (a *App) GetFilterByUser() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.filterByUser
}

// fetchAndUpdateIncidents fetches incidents from PagerDuty and updates the database
func (a *App) fetchAndUpdateIncidents() {
	if a.client == nil {
		return
	}

	// Get selected services and filter state with read lock
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	shouldFilterByUser := a.filterByUser
	a.mu.RUnlock()

	// Determine user ID if filtering by user
	var userID string
	if shouldFilterByUser {
		user, err := a.client.GetCurrentUser()
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to get current user: %v", err))
			return
		}
		userID = user.ID
	}

	// Fetch open incidents (both triggered and acknowledged)
	incidents, err := a.client.FetchOpenIncidents(selectedServices, userID)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch open incidents: %v", err))
		return
	}

	// Also fetch recently resolved incidents for immediate status updates
	// This ensures that when an incident moves from open to resolved,
	// it appears immediately in the resolved tab
	resolvedOpts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      time.Now().Add(-24 * time.Hour), // Last 24 hours for immediate updates
	}

	resolvedIncidents, err := a.client.FetchIncidentsWithOptions(resolvedOpts)
	if err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to fetch recent resolved incidents: %v", err))
		// Don't return here, continue with open incidents update
	}

	// Update database with all incidents
	updatedCount := 0
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert incident: %v", err))
		} else {
			updatedCount++
		}
	}

	// Update resolved incidents
	for _, incident := range resolvedIncidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		} else {
			updatedCount++
		}
	}

	if updatedCount > 0 {
		a.logger.Debug(fmt.Sprintf("Updated %d incidents in database", updatedCount))
	}

	// Emit event to update both open and resolved views
	runtime.EventsEmit(a.ctx, "incidents-updated", "both")
	// After updating database, check for triggered incidents
	openIncidents, err := a.db.GetOpenIncidents()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to get open incidents for notification check: %v", err))
		return
	}

	// Use dedicated mutex for lastIncidents
	a.lastIncidentsMu.Lock()
	defer a.lastIncidentsMu.Unlock()

	for _, incident := range openIncidents {
		lastStatus, exists := a.lastIncidents[incident.IncidentID]

		// Check if this is a new triggered incident or status changed to triggered
		if incident.Status == "triggered" && (!exists || lastStatus != "triggered") {
			// Send notification for triggered incident
			if a.notificationMgr != nil {
				err := a.notificationMgr.SendNotification(
					incident.ServiceSummary,
					incident.Title,
					incident.ServiceSummary,
				)
				if err != nil {
					a.logger.Error(fmt.Sprintf("Failed to send notification: %v", err))
				}
				a.logger.Info(fmt.Sprintf("Notification sent for triggered incident: %s", incident.IncidentID))
			}
		}

		// Update last known status
		a.lastIncidents[incident.IncidentID] = incident.Status
	}

	// Clean up resolved incidents from tracking
	incidentMap := make(map[string]bool)
	for _, incident := range openIncidents {
		incidentMap[incident.IncidentID] = true
	}

	for id := range a.lastIncidents {
		if !incidentMap[id] {
			delete(a.lastIncidents, id)
		}
	}
}

// StartPolling starts the incident polling
func (a *App) StartPolling() {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	if a.polling {
		return
	}

	a.polling = true
	a.pollTicker = time.NewTicker(3 * time.Second)
	a.logger.Info("Started incident polling (3s interval)")

	go func() {
		// Initial fetch immediately
		a.fetchAndUpdateIncidents()

		for range a.pollTicker.C {
			// Check polling state with lock
			a.pollMu.RLock()
			shouldContinue := a.polling
			a.pollMu.RUnlock()

			if !shouldContinue {
				break
			}
			a.fetchAndUpdateIncidents()
		}
	}()
}

// StopPolling stops the incident polling
func (a *App) StopPolling() {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	a.polling = false
	if a.pollTicker != nil {
		a.pollTicker.Stop()
		a.pollTicker = nil
	}
	a.logger.Info("Stopped incident polling")
}

// GetOpenIncidents returns open incidents filtered by selected services
func (a *App) GetOpenIncidents(serviceIDs []string) ([]database.IncidentData, error) {
	if a.db == nil {
		err := fmt.Errorf("database not initialized")
		a.logger.Error(err.Error())
		return nil, err
	}

	// Don't fetch if polling is active - just return cached data
	// The polling mechanism is already updating the database every 3 seconds
	a.pollMu.RLock()
	isPolling := a.polling
	a.pollMu.RUnlock()

	// Only fetch manually if polling is not active
	if !isPolling && a.client != nil {
		a.fetchAndUpdateIncidents()
	}

	// Get all open incidents from database
	allIncidents, err := a.db.GetOpenIncidents()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to get open incidents: %v", err))
		return nil, err
	}

	// If no services selected, return all
	if len(serviceIDs) == 0 {
		return allIncidents, nil
	}

	// Filter by selected services
	serviceMap := make(map[string]bool)
	for _, id := range serviceIDs {
		serviceMap[id] = true
	}

	var filteredIncidents []database.IncidentData
	for _, incident := range allIncidents {
		if serviceMap[incident.ServiceID] {
			filteredIncidents = append(filteredIncidents, incident)
		}
	}

	return filteredIncidents, nil
}

// GetResolvedIncidents fetches and returns resolved incidents for selected services
func (a *App) GetResolvedIncidents(serviceIDs []string) ([]database.IncidentData, error) {
	if a.client == nil {
		err := fmt.Errorf("PagerDuty client not initialized")
		a.logger.Warn(err.Error())
		return nil, err
	}

	// Only fetch if we have services configured
	if len(serviceIDs) == 0 {
		a.logger.Info("No services selected, returning empty resolved incidents")
		return []database.IncidentData{}, nil
	}

	// Check if we have cached resolved incidents for these services
	cachedIncidents, err := a.db.GetResolvedIncidentsByServices(serviceIDs)
	if err == nil && len(cachedIncidents) > 0 {
		// Return cached data immediately for faster load
		go func() {
			// Fetch fresh data in background
			incidents, err := a.client.FetchResolvedIncidents(serviceIDs)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
				return
			}
			// Update database
			for _, incident := range incidents {
				if err := a.db.UpsertIncident(incident); err != nil {
					a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
				}
			}

			// Emit event to update UI
			runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
		}()
		return cachedIncidents, nil
	}

	// No cache, fetch from PagerDuty
	incidents, err := a.client.FetchResolvedIncidents(serviceIDs)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
		return nil, fmt.Errorf("failed to fetch resolved incidents: %w", err)
	}

	// Update database
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}
	}

	// Return filtered incidents
	return a.db.GetResolvedIncidentsByServices(serviceIDs)
}

// ConfigureAPIKey saves the API key and initializes the PagerDuty client
func (a *App) ConfigureAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Initialize PagerDuty client
	client, err := store.NewClient(apiKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to create PagerDuty client: %v", err))
		return fmt.Errorf("failed to create PagerDuty client: %w", err)
	}

	// Test the API key by getting current user
	_, err = client.GetCurrentUser()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to validate API key: %v", err))
		return fmt.Errorf("invalid API key: %w", err)
	}

	// Save to keyring if available
	if a.kr != nil {
		if err := a.kr.Set(keyring.Item{
			Key:  "pagerduty-api-key",
			Data: []byte(apiKey),
		}); err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to save API key to keyring: %v", err))
			// Continue even if keyring save fails
		}
	}

	// Update client
	a.client = client
	a.logger.Info("API key configured successfully")

	// Start polling with new client
	a.StartPolling()

	// Emit event to notify UI
	runtime.EventsEmit(a.ctx, "api-key-configured")

	return nil
}

// GetAPIKey retrieves the stored API key
func (a *App) GetAPIKey() (string, error) {
	if a.kr == nil {
		return "", fmt.Errorf("keyring not available")
	}

	item, err := a.kr.Get("pagerduty-api-key")
	if err != nil {
		return "", err
	}

	return string(item.Data), nil
}

// UploadServicesConfig processes uploaded services configuration
func (a *App) UploadServicesConfig(jsonData string) error {
	var config store.ServicesConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to parse services config: %v", err))
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.servicesConfig = &config
	a.selectedServices = []string{}

	// Auto-select all services
	for _, service := range config.Services {
		switch id := service.ID.(type) {
		case string:
			a.selectedServices = append(a.selectedServices, id)
		case []interface{}:
			for _, serviceID := range id {
				if strID, ok := serviceID.(string); ok {
					a.selectedServices = append(a.selectedServices, strID)
				}
			}
		case float64:
			// Handle numeric IDs that come from JSON
			a.selectedServices = append(a.selectedServices, fmt.Sprintf("%.0f", id))
		}
	}

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "services-config-updated")

	return nil
}

// RemoveServicesConfig removes the current services configuration
func (a *App) RemoveServicesConfig() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.servicesConfig = nil
	a.selectedServices = []string{}

	a.logger.Info("Services configuration removed")

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "services-config-updated")

	return nil
}

// GetServicesConfig returns the current services configuration
func (a *App) GetServicesConfig() (*store.ServicesConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.servicesConfig == nil {
		return nil, fmt.Errorf("no services configuration loaded")
	}
	return a.servicesConfig, nil
}

// SetSelectedServices updates the selected services for filtering
func (a *App) SetSelectedServices(services []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.selectedServices = services
	if a.logger != nil {
		a.logger.Debug(fmt.Sprintf("Selected services updated: %d services", len(services)))
	}
}

// GetSelectedServices returns the currently selected services
func (a *App) GetSelectedServices() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]string{}, a.selectedServices...)
}

// ReadFile reads a file and returns its content
func (a *App) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to read file %s: %v", path, err))
		return "", err
	}
	return string(content), nil
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	a.StopPolling()
	a.StopResolvedPolling()

	// Close database
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to close database: %v", err))
		}
	}

	a.logger.Info("PagerOps shutting down...")
}

func (a *App) GetNotificationConfig() NotificationConfig {
	if a.notificationMgr == nil {
		return NotificationConfig{}
	}
	return a.notificationMgr.GetConfig()
}

func (a *App) SetNotificationEnabled(enabled bool) {
	if a.notificationMgr != nil {
		a.notificationMgr.SetEnabled(enabled)
	}
}

func (a *App) SetNotificationSound(sound string) {
	if a.notificationMgr != nil {
		a.notificationMgr.SetSound(sound)
	}
}

func (a *App) SnoozeNotificationSound(minutes int) {
	if a.notificationMgr != nil {
		a.notificationMgr.SnoozeSound(minutes)
		runtime.EventsEmit(a.ctx, "notification-snoozed", minutes)
	}
}

func (a *App) UnsnoozeNotificationSound() {
	if a.notificationMgr != nil {
		a.notificationMgr.UnsnoozeSound()
		runtime.EventsEmit(a.ctx, "notification-unsnoozed")
	}
}

func (a *App) GetAvailableSounds() ([]string, error) {
	if a.notificationMgr == nil {
		return []string{"default"}, nil
	}
	return a.notificationMgr.GetAvailableSounds()
}

func (a *App) TestNotificationSound() error {
	if a.notificationMgr == nil {
		return fmt.Errorf("notification manager not initialized")
	}

	// Get current sound setting and ensure it has proper extension if needed
	config := a.notificationMgr.GetConfig()
	if config.Sound != "default" && !strings.Contains(config.Sound, ".") {
		// Find the actual file with extension
		soundsDir := filepath.Join(".", "assets", "sounds")
		entries, err := os.ReadDir(soundsDir)
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
				if nameWithoutExt == config.Sound {
					// Temporarily set the full filename for testing
					a.notificationMgr.mu.Lock()
					originalSound := a.notificationMgr.config.Sound
					a.notificationMgr.config.Sound = name
					a.notificationMgr.mu.Unlock()

					err := a.notificationMgr.TestSound()

					// Restore original setting
					a.notificationMgr.mu.Lock()
					a.notificationMgr.config.Sound = originalSound
					a.notificationMgr.mu.Unlock()

					return err
				}
			}
		}
	}

	return a.notificationMgr.TestSound()
}

func (a *App) IsNotificationSnoozed() bool {
	if a.notificationMgr == nil {
		return false
	}
	return a.notificationMgr.IsSnoozeActive()
}

// StartResolvedPolling starts polling for resolved incidents only
func (a *App) StartResolvedPolling() {
	a.resolvedPollMu.Lock()
	defer a.resolvedPollMu.Unlock()

	if a.resolvedPolling {
		return
	}

	a.resolvedPolling = true
	a.resolvedPollTicker = time.NewTicker(30 * time.Second) // 30 second interval for resolved
	a.logger.Info("Started resolved incidents polling (30s interval)")

	go func() {
		// Initial fetch
		a.fetchResolvedIncidents()

		for range a.resolvedPollTicker.C {
			a.resolvedPollMu.RLock()
			shouldContinue := a.resolvedPolling
			a.resolvedPollMu.RUnlock()

			if !shouldContinue {
				break
			}

			// Check rate limit before making call
			if a.rateLimitTracker.CanMakeCall() {
				a.fetchResolvedIncidents()
				a.rateLimitTracker.RecordCall()

				// Log rate limit status every 10 calls
				currentRate := a.rateLimitTracker.GetCurrentRate()
				if currentRate%10 == 0 {
					a.logger.Debug(fmt.Sprintf("Rate limit status: %d/960 calls per minute", currentRate))
				}
			} else {
				a.logger.Warn("Rate limit approaching, skipping resolved incidents fetch")
			}
		}
	}()
}

// StopResolvedPolling stops the resolved incidents polling
func (a *App) StopResolvedPolling() {
	a.resolvedPollMu.Lock()
	defer a.resolvedPollMu.Unlock()

	a.resolvedPolling = false
	if a.resolvedPollTicker != nil {
		a.resolvedPollTicker.Stop()
		a.resolvedPollTicker = nil
	}
	a.logger.Info("Stopped resolved incidents polling")
}

// fetchResolvedIncidents fetches only resolved incidents
func (a *App) fetchResolvedIncidents() {
	if a.client == nil {
		return
	}

	a.mu.RLock()
	selectedServices := a.selectedServices
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Fetch resolved incidents from last 48 hours
	resolvedOpts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      time.Now().Add(-48 * time.Hour),
	}

	incidents, err := a.client.FetchIncidentsWithOptions(resolvedOpts)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
		return
	}

	// Update database
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}
	}

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
}

// GetRateLimitStatus returns the current rate limit status
func (a *App) GetRateLimitStatus() map[string]interface{} {
	currentRate := a.rateLimitTracker.GetCurrentRate()
	return map[string]interface{}{
		"current":    currentRate,
		"max":        960,
		"remaining":  960 - currentRate,
		"percentage": float64(currentRate) / 960.0 * 100,
	}
}
