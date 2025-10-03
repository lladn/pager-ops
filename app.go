package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"pager-ops/database"
	"pager-ops/store"

	"github.com/99designs/keyring"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct - NO FIELDS RENAMED, only new fields added
type App struct {
	ctx                   context.Context
	db                    *database.DB
	client                *store.Client
	polling               bool
	pollTicker            *time.Ticker
	servicesConfig        *store.ServicesConfig
	selectedServices      []string
	kr                    keyring.Keyring
	logger                *Logger
	filterByUser          bool
	mu                    sync.RWMutex
	pollMu                sync.RWMutex
	notificationMgr       *NotificationManager
	lastIncidents         map[string]string
	lastIncidentsMu       sync.RWMutex
	resolvedPollTicker    *time.Ticker
	resolvedPolling       bool
	resolvedPollMu        sync.RWMutex
	rateLimitTracker      *RateLimitTracker
	userCache             *UserCache
	lastResolvedFetch     time.Time
	lastResolvedFetchMu   sync.RWMutex
	circuitBreaker        *CircuitBreaker
	previousOpenIncidents map[string]database.IncidentData
	previousOpenMu        sync.RWMutex
	shutdownChan          chan struct{}
	shutdownWg            sync.WaitGroup
	userPolling           bool
	userPollTicker        *time.Ticker
	userPollMu            sync.RWMutex
	latestResolvedDate    time.Time
	latestResolvedMu      sync.RWMutex
	resolvedFetchMu       sync.Mutex
	sidebarFetchingMu     sync.Mutex
	fetchingIncidents     map[string]bool
}

// RateLimitTracker
type RateLimitTracker struct {
	mu         sync.Mutex
	calls      []time.Time
	windowSize time.Duration
	maxCalls   int
}

// New structures added
type UserCache struct {
	user      interface{}
	userID    string
	expiresAt time.Time
	mu        sync.RWMutex
}

type CircuitBreaker struct {
	failures          int32
	lastFailure       time.Time
	state             int32 // 0: closed, 1: open, 2: half-open
	maxFailures       int32
	cooldownPeriod    time.Duration
	backoffMultiplier float64
	currentBackoff    time.Duration
	mu                sync.RWMutex
}

func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		windowSize: time.Minute,
		maxCalls:   864, // 90% of 960 per minute limit
		calls:      make([]time.Time, 0),
	}
}

func NewUserCache() *UserCache {
	return &UserCache{
		expiresAt: time.Now(),
	}
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:       5,
		cooldownPeriod:    30 * time.Second,
		backoffMultiplier: 2.0,
		currentBackoff:    30 * time.Second,
	}
}

// RateLimitTracker methods
func (r *RateLimitTracker) CanMakeCall() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	// Remove calls outside the window
	validCalls := []time.Time{}
	for _, callTime := range r.calls {
		if callTime.After(cutoff) {
			validCalls = append(validCalls, callTime)
		}
	}
	r.calls = validCalls

	return len(r.calls) < r.maxCalls
}

func (r *RateLimitTracker) RecordCall() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, time.Now())
}

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

func (cb *CircuitBreaker) Allow() bool {
	state := atomic.LoadInt32(&cb.state)

	if state == 0 { // Closed
		return true
	}

	if state == 1 { // Open
		cb.mu.RLock()
		lastFailure := cb.lastFailure
		currentBackoff := cb.currentBackoff
		cb.mu.RUnlock()

		if time.Since(lastFailure) > currentBackoff {
			// Try half-open state
			atomic.StoreInt32(&cb.state, 2)
			return true
		}
		return false
	}

	return true // Half-open
}

func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, 0) // Closed

	// Reset backoff on success
	cb.mu.Lock()
	cb.currentBackoff = cb.cooldownPeriod
	cb.mu.Unlock()
}

func (cb *CircuitBreaker) RecordFailure() {
	failures := atomic.AddInt32(&cb.failures, 1)

	cb.mu.Lock()
	cb.lastFailure = time.Now()

	// Exponential backoff: double the backoff period on each failure
	if failures >= cb.maxFailures {
		atomic.StoreInt32(&cb.state, 1) // Open

		// Increase backoff exponentially, cap at 5 minutes
		cb.currentBackoff = time.Duration(float64(cb.currentBackoff) * cb.backoffMultiplier)
		if cb.currentBackoff > 5*time.Minute {
			cb.currentBackoff = 5 * time.Minute
		}
	}
	cb.mu.Unlock()
}

func (uc *UserCache) Get() (string, bool) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	if time.Now().After(uc.expiresAt) {
		return "", false
	}

	return uc.userID, uc.userID != ""
}

func (uc *UserCache) Set(userID string, user interface{}) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.userID = userID
	uc.user = user
	uc.expiresAt = time.Now().Add(1 * time.Hour)
}

func (uc *UserCache) Invalidate() {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.userID = ""
	uc.user = nil
	uc.expiresAt = time.Time{}
}

func NewApp() *App {
	return &App{
		filterByUser:          true,
		lastIncidents:         make(map[string]string),
		previousOpenIncidents: make(map[string]database.IncidentData),
		shutdownChan:          make(chan struct{}),
		latestResolvedDate:    time.Now().Add(-72 * time.Hour), // Initialize to 3 days ago
		fetchingIncidents:     make(map[string]bool),
	}
}

func (a *App) startup(
	ctx context.Context,
) {
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

	// Initialize state table for persistence
	if err := a.db.InitStateTable(); err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to initialize state table: %v", err))
	}

	// Load latest resolved date from database
	if timestamp, err := a.db.GetState("latest_resolved_date"); err == nil && timestamp != "" {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			a.latestResolvedMu.Lock()
			a.latestResolvedDate = t
			a.latestResolvedMu.Unlock()
			a.logger.Info(fmt.Sprintf("Restored latest resolved date: %s", timestamp))
		}
	}

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

	// Load browser redirect setting from database
	if a.db != nil {
		if value, err := a.db.GetState("browser_redirect"); err == nil {
			if value == "true" && a.notificationMgr != nil {
				a.notificationMgr.SetBrowserRedirect(true)
				a.logger.Info("Browser redirect enabled from saved settings")
			}
		}
	}

	// Initialize production components
	a.rateLimitTracker = NewRateLimitTracker()
	a.userCache = NewUserCache()
	a.circuitBreaker = NewCircuitBreaker()

	// Start sidebar data cleanup routine
	go a.cleanupOldSidebarData()

	// In the startup method, modify the section where API key is loaded:
	// Try to load API key and initialize client
	apiKey, err := a.GetAPIKey()
	if err == nil && apiKey != "" {
		client, err := store.NewClient(apiKey)
		if err == nil {
			a.client = client
			a.logger.Info("PagerDuty client initialized successfully")

			// Fetch and cache user ID on startup
			if user, err := client.GetCurrentUser(); err == nil {
				if a.userCache == nil {
					a.userCache = NewUserCache()
				}
				a.userCache.Set(user.ID, user)
				a.logger.Info(fmt.Sprintf("Cached user ID on startup: %s", user.ID))
			}

			// Set default filter to true (show only assigned)
			a.filterByUser = true

			// Start all three polling mechanisms
			a.StartPolling()
			a.StartUserPolling()
			a.StartResolvedPolling()

			// Perform initial resolved fetch
			go a.performInitialResolvedFetch()
		} else {
			a.logger.Warn(fmt.Sprintf("Failed to initialize PagerDuty client: %v", err))
		}
	}
}

func (a *App) SetFilterByUser(
	enabled bool,
) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filterByUser = enabled
	a.logger.Info(fmt.Sprintf("Filter by user set to: %v", enabled))

	// Trigger immediate update
	if a.polling {
		go a.fetchAndUpdateIncidents()
	}
}

func (a *App) GetFilterByUser() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.filterByUser
}

func (a *App) fetchAndUpdateIncidents() {
	// This method now serves as a unified update trigger
	a.mu.RLock()
	shouldFilterByUser := a.filterByUser
	a.mu.RUnlock()

	if shouldFilterByUser {
		a.fetchUserIncidents()
	} else {
		a.fetchServiceIncidents()
	}
}

func (a *App) processAndUpdateIncidents(
	incidents []database.IncidentData,
	source string,
) {
	// Check shutdown before database operations
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Get selected services for filtering
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	// Collect incident IDs from current fetch
	currentIncidentIDs := make([]string, len(incidents))
	for i, incident := range incidents {
		currentIncidentIDs[i] = incident.IncidentID
	}

	// Get all currently open incidents from database before update
	existingOpenIncidents, err := a.db.GetOpenIncidents()
	if err != nil {
		if err.Error() == "sql: database is closed" {
			return
		}
		a.logger.Error(fmt.Sprintf("Failed to get existing open incidents: %v", err))
		return
	}

	// Identify stale incidents (in DB but not in API response)
	staleIDs := []string{}
	existingMap := make(map[string]bool)
	for _, existing := range existingOpenIncidents {
		existingMap[existing.IncidentID] = true
	}

	currentMap := make(map[string]bool)
	for _, incident := range incidents {
		currentMap[incident.IncidentID] = true
	}

	// Find incidents that are in DB but not in current API response
	for _, existing := range existingOpenIncidents {
		if !currentMap[existing.IncidentID] {
			// Only mark as stale if it's from the same service set we're fetching
			if len(selectedServices) == 0 || containsService(selectedServices, existing.ServiceID) {
				staleIDs = append(staleIDs, existing.IncidentID)
				a.logger.Info(fmt.Sprintf("[%s] Marking incident as resolved (not in API): %s", source, existing.IncidentID))
			}
		}
	}

	// Use batch update for better atomicity
	if err := a.db.UpdateIncidentsBatch(incidents, staleIDs); err != nil {
		if err.Error() == "sql: database is closed" {
			a.logger.Info("Database closed, stopping incident updates")
			return
		}
		a.logger.Error(fmt.Sprintf("Failed to batch update incidents: %v", err))
		// Fall back to individual updates
		for _, incident := range incidents {
			if err := a.db.UpsertIncident(incident); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to upsert incident: %v", err))
			}
		}
		// Still try to remove stale incidents
		if len(currentIncidentIDs) > 0 || len(selectedServices) > 0 {
			if err := a.db.RemoveStaleOpenIncidents(currentIncidentIDs, selectedServices); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to remove stale incidents: %v", err))
			}
		}
	}

	// Get updated open incidents after database changes
	allOpenIncidents, err := a.db.GetOpenIncidents()
	if err != nil {
		if err.Error() == "sql: database is closed" {
			return
		}
		a.logger.Error(fmt.Sprintf("Failed to get all open incidents: %v", err))
		return
	}

	currentOpen := make(map[string]database.IncidentData)
	for _, incident := range allOpenIncidents {
		currentOpen[incident.IncidentID] = incident
	}

	// Get previous state with proper locking
	a.previousOpenMu.RLock()
	previousOpen := make(map[string]database.IncidentData)
	for k, v := range a.previousOpenIncidents {
		previousOpen[k] = v
	}
	a.previousOpenMu.RUnlock()

	// Detect REAL status transitions
	var hasTransitions bool
	for id, prevIncident := range previousOpen {
		if _, exists := currentOpen[id]; !exists {
			// Incident truly moved from open to resolved
			a.logger.Info(fmt.Sprintf("[%s] Detected transition to resolved: %s", source, id))
			hasTransitions = true
		} else if currentOpen[id].Status != prevIncident.Status {
			// Status changed within open states
			a.logger.Info(fmt.Sprintf("[%s] Status change for %s: %s -> %s",
				source, id, prevIncident.Status, currentOpen[id].Status))
		}
	}

	// Log new incidents that appeared
	for id := range currentOpen {
		if _, existed := previousOpen[id]; !existed {
			a.logger.Debug(fmt.Sprintf("[%s] New incident detected: %s", source, id))
		}
	}

	// If transitions detected, trigger lightweight resolved fetch
	if hasTransitions {
		a.logger.Info(fmt.Sprintf("[%s] Transitions detected, resolved polling will update", source))
	}

	// Update previous state with proper locking
	a.previousOpenMu.Lock()
	a.previousOpenIncidents = currentOpen
	a.previousOpenMu.Unlock()

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "incidents-updated", "both")

	// Check for triggered incidents and send notifications
	a.checkForTriggeredIncidents()
}

func containsService(services []string, serviceID string) bool {
	for _, s := range services {
		if s == serviceID {
			return true
		}
	}
	return false
}

func (a *App) checkForTriggeredIncidents() {
	openIncidents, err := a.db.GetOpenIncidents()
	if err != nil {
		if err.Error() == "sql: database is closed" {
			return
		}
		a.logger.Error(fmt.Sprintf("Failed to get open incidents for notification check: %v", err))
		return
	}

	// Get selected services to filter notifications
	a.mu.RLock()
	selectedServices := make([]string, len(a.selectedServices))
	copy(selectedServices, a.selectedServices)
	a.mu.RUnlock()

	// Use dedicated mutex for lastIncidents
	a.lastIncidentsMu.Lock()
	defer a.lastIncidentsMu.Unlock()

	for _, incident := range openIncidents {
		// Skip notifications for incidents from non-selected services
		if len(selectedServices) > 0 && !containsService(selectedServices, incident.ServiceID) {
			// Still track the status for when the service is re-selected
			a.lastIncidents[incident.IncidentID] = incident.Status
			continue
		}

		lastStatus, exists := a.lastIncidents[incident.IncidentID]

		// Check if this is a new triggered incident or status changed to triggered
		if incident.Status == "triggered" && (!exists || lastStatus != "triggered") {
			// Get the configured service name for the say command
			serviceName := a.GetServiceNameByID(incident.ServiceID)
			if serviceName == "" {
				// Fallback to service summary if no configured name found
				serviceName = incident.ServiceSummary
			}

			// Send notification for triggered incident
			if a.notificationMgr != nil {
				err := a.notificationMgr.SendNotification(
					incident.ServiceSummary, // Title for terminal-notifier
					incident.Title,          // Message for terminal-notifier
					incident.HTMLURL,        // URL for click-to-open
					serviceName,             // Service name for say command
				)
				if err != nil {
					a.logger.Error(fmt.Sprintf("Failed to send notification: %v", err))
				}
				a.logger.Info(fmt.Sprintf("Notification sent for triggered incident: %s (service: %s)",
					incident.IncidentID, serviceName))

				// Queue browser redirect if enabled
				a.notificationMgr.QueueBrowserRedirect(incident.IncidentID, incident.HTMLURL)
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

	// Remove resolved incidents from tracking - already protected by defer
	for id := range a.lastIncidents {
		if !incidentMap[id] {
			delete(a.lastIncidents, id)
		}
	}
}

func (a *App) SetBrowserRedirect(enabled bool) {
	if a.notificationMgr != nil {
		a.notificationMgr.SetBrowserRedirect(enabled)

		// Persist the setting
		if a.db != nil {
			value := "false"
			if enabled {
				value = "true"
			}
			if err := a.db.SetState("browser_redirect", value); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to persist browser redirect setting: %v", err))
			}
		}
	}
}

func (a *App) GetBrowserRedirect() bool {
	if a.notificationMgr != nil {
		config := a.notificationMgr.GetConfig()
		return config.BrowserRedirect
	}
	return false
}

func (a *App) StartPolling() {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	if a.polling {
		return
	}

	a.polling = true
	a.pollTicker = time.NewTicker(3 * time.Second)
	a.logger.Info("Started service incidents polling (3s interval)")

	// Store ticker channel reference while holding lock
	tickerChan := a.pollTicker.C

	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()

		// Initial fetch immediately
		a.fetchServiceIncidents()

		for {
			select {
			case <-a.shutdownChan:
				a.logger.Info("Service incidents polling stopped by shutdown signal")
				return
			case <-tickerChan:
				// Check polling state with lock
				a.pollMu.RLock()
				shouldContinue := a.polling
				currentTicker := a.pollTicker
				a.pollMu.RUnlock()

				if !shouldContinue || currentTicker == nil {
					return
				}

				// Check rate limit before making call
				if !a.rateLimitTracker.CanMakeCall() {
					a.logger.Warn("Rate limit approaching threshold, skipping service fetch")
					continue
				}

				a.fetchServiceIncidents()
				a.rateLimitTracker.RecordCall()
			}
		}
	}()
}

func (a *App) StartUserPolling() {
	a.userPollMu.Lock()
	defer a.userPollMu.Unlock()

	if a.userPolling {
		return
	}

	a.userPolling = true
	a.userPollTicker = time.NewTicker(6 * time.Second)
	a.logger.Info("Started user incidents polling (6s interval)")

	// Store ticker channel reference while holding lock
	tickerChan := a.userPollTicker.C

	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()

		// Initial fetch immediately if filter is enabled
		a.mu.RLock()
		shouldFetch := a.filterByUser
		a.mu.RUnlock()

		if shouldFetch {
			a.fetchUserIncidents()
		}

		for {
			select {
			case <-a.shutdownChan:
				a.logger.Info("User incidents polling stopped by shutdown signal")
				return
			case <-tickerChan:
				// Check polling state with lock
				a.userPollMu.RLock()
				shouldContinue := a.userPolling
				currentTicker := a.userPollTicker
				a.userPollMu.RUnlock()

				if !shouldContinue || currentTicker == nil {
					return
				}

				// Check if user filtering is enabled
				a.mu.RLock()
				shouldFetch := a.filterByUser
				a.mu.RUnlock()

				if !shouldFetch {
					continue // Skip if user filtering is disabled
				}

				// Check rate limit before making call
				if !a.rateLimitTracker.CanMakeCall() {
					a.logger.Warn("Rate limit approaching threshold, skipping user fetch")
					continue
				}

				a.fetchUserIncidents()
				a.rateLimitTracker.RecordCall()
			}
		}
	}()
}

func (a *App) StopUserPolling() {
	a.userPollMu.Lock()
	defer a.userPollMu.Unlock()

	a.userPolling = false
	if a.userPollTicker != nil {
		a.userPollTicker.Stop()
		a.userPollTicker = nil
	}
	a.logger.Info("Stopped user incidents polling")
}

// StopPolling - Original method unchanged
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

func (a *App) StartResolvedPolling() {
	a.resolvedPollMu.Lock()
	defer a.resolvedPollMu.Unlock()

	if a.resolvedPolling {
		return
	}

	a.resolvedPolling = true
	a.resolvedPollTicker = time.NewTicker(1 * time.Minute) // Changed from 10 minutes to 1 minute
	a.logger.Info("Started resolved incidents polling (1m interval)")

	// Store ticker channel reference while holding lock
	tickerChan := a.resolvedPollTicker.C

	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()

		// Initial fetch using new method
		a.fetchResolvedIncidentsSince()

		for {
			select {
			case <-a.shutdownChan:
				a.logger.Info("Resolved incidents polling stopped by shutdown signal")
				return
			case <-tickerChan:
				a.resolvedPollMu.RLock()
				shouldContinue := a.resolvedPolling
				currentTicker := a.resolvedPollTicker
				a.resolvedPollMu.RUnlock()

				if !shouldContinue || currentTicker == nil {
					return
				}

				// Check rate limit before making call
				if a.rateLimitTracker.CanMakeCall() {
					a.fetchResolvedIncidentsSince()
					a.rateLimitTracker.RecordCall()

					// Log rate limit status periodically
					currentRate := a.rateLimitTracker.GetCurrentRate()
					if currentRate%10 == 0 {
						a.logger.Debug(fmt.Sprintf("Rate limit status: %d/960 calls per minute", currentRate))
					}
				} else {
					a.logger.Warn("Rate limit approaching, skipping resolved incidents fetch")
				}
			}
		}
	}()
}

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

func (a *App) fetchServiceIncidents() {
	if a.client == nil {
		return
	}

	// Check if shutdown is in progress
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Check circuit breaker
	if !a.circuitBreaker.Allow() {
		a.logger.Warn("Circuit breaker open, skipping service fetch")
		return
	}

	// Get selected services with proper locking
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		// Still need to clear any stale open incidents
		if err := a.db.RemoveStaleOpenIncidents([]string{}, []string{}); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to clear stale incidents: %v", err))
		}
		return
	}

	// Fetch open incidents for services WITHOUT user filtering
	incidents, err := a.fetchWithRetry(func() ([]database.IncidentData, error) {
		return a.client.FetchOpenIncidents(selectedServices, "")
	}, 3)

	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch service incidents after retries: %v", err))
		a.circuitBreaker.RecordFailure()
		return
	}

	a.circuitBreaker.RecordSuccess()
	a.processAndUpdateIncidents(incidents, "services")
}

func (a *App) fetchUserIncidents() {
	if a.client == nil {
		return
	}

	// Check if shutdown is in progress
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Check circuit breaker
	if !a.circuitBreaker.Allow() {
		a.logger.Warn("Circuit breaker open, skipping user fetch")
		return
	}

	// Get or refresh user ID with caching
	cachedID, valid := a.userCache.Get()
	var userID string

	if valid {
		userID = cachedID
	} else {
		// Fetch user asynchronously to avoid blocking
		go a.refreshUserCache()

		// Try to get current user synchronously for this cycle
		if user, err := a.client.GetCurrentUser(); err == nil {
			userID = user.ID
			a.userCache.Set(userID, user)
		} else {
			a.logger.Error(fmt.Sprintf("Failed to get current user: %v", err))
			a.circuitBreaker.RecordFailure()
			return
		}
	}

	if userID == "" {
		return
	}

	// Get selected services for context
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	// Fetch incidents assigned to user (API already filters by services if provided)
	incidents, err := a.fetchWithRetry(func() ([]database.IncidentData, error) {
		return a.client.FetchOpenIncidents(selectedServices, userID)
	}, 3)

	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch user incidents after retries: %v", err))
		a.circuitBreaker.RecordFailure()
		return
	}

	a.circuitBreaker.RecordSuccess()
	a.processAndUpdateIncidents(incidents, "user")
}

func (a *App) fetchResolvedIncidentsSince() {
	if a.client == nil || !a.circuitBreaker.Allow() {
		return
	}

	// Prevent concurrent resolved fetches
	if !a.resolvedFetchMu.TryLock() {
		a.logger.Debug("Skipping resolved fetch - another fetch in progress")
		return
	}
	defer a.resolvedFetchMu.Unlock()

	// Check if shutdown is in progress
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Get the latest resolved date
	a.latestResolvedMu.RLock()
	since := a.latestResolvedDate
	a.latestResolvedMu.RUnlock()

	now := time.Now()

	// Safety check: don't go back more than 7 days
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	if since.Before(sevenDaysAgo) {
		since = sevenDaysAgo
		a.logger.Info("Limiting resolved fetch to 7 days for performance")
	}

	resolvedOpts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      since,
		Until:      now,
	}

	// Use paginated fetch with smaller page size to reduce timeout risk
	incidents, err := a.client.FetchIncidentsWithPagination(resolvedOpts, 50)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
		a.circuitBreaker.RecordFailure()
		return
	}

	a.circuitBreaker.RecordSuccess()

	// Check shutdown before database operations
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Update database and track latest date
	var latestDate time.Time
	updateCount := 0
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			if err.Error() == "sql: database is closed" {
				a.logger.Info("Database closed, stopping resolved incident updates")
				return
			}
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		} else {
			updateCount++
			if incident.UpdatedAt.After(latestDate) {
				latestDate = incident.UpdatedAt
			}
		}
	}

	// Update latest resolved date if newer
	if !latestDate.IsZero() {
		a.latestResolvedMu.Lock()
		if latestDate.After(a.latestResolvedDate) {
			a.latestResolvedDate = latestDate
			// Persist to database
			if err := a.db.SetState("latest_resolved_date", latestDate.Format(time.RFC3339)); err != nil {
				a.logger.Warn(fmt.Sprintf("Failed to persist latest resolved date: %v", err))
			}
		}
		a.latestResolvedMu.Unlock()
	}

	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
}

// New adaptive fetching method
func (a *App) fetchResolvedIncidentsAdaptive() {
	if a.client == nil || !a.circuitBreaker.Allow() {
		return
	}

	// Check if shutdown is in progress
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Calculate adaptive window based on gap
	a.lastResolvedFetchMu.RLock()
	lastFetch := a.lastResolvedFetch
	a.lastResolvedFetchMu.RUnlock()

	var since time.Time
	now := time.Now()
	gap := now.Sub(lastFetch)

	switch {
	case gap <= 15*time.Minute:
		since = now.Add(-6 * time.Hour) // Standard window
	case gap <= 48*time.Hour:
		since = now.Add(-(gap + 30*time.Minute)) // Dynamic window with overlap
	default:
		since = now.Add(-72 * time.Hour) // Full safety window
	}

	a.logger.Info(fmt.Sprintf("Adaptive fetch window: %v (gap: %v)", now.Sub(since), gap))

	resolvedOpts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      since,
		Until:      now,
	}

	// Use paginated fetch ONLY for resolved incidents
	incidents, err := a.client.FetchIncidentsWithPagination(resolvedOpts, 100)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
		a.circuitBreaker.RecordFailure()
		return
	}

	a.circuitBreaker.RecordSuccess()

	// Check shutdown before database operations
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Update database
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			if err.Error() == "sql: database is closed" {
				a.logger.Info("Database closed, stopping resolved incident updates")
				return
			}
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}
	}

	// Update last fetch timestamp
	a.lastResolvedFetchMu.Lock()
	a.lastResolvedFetch = now
	a.lastResolvedFetchMu.Unlock()

	// Persist to database
	if err := a.db.SetState("last_resolved_fetch", now.Format(time.RFC3339)); err != nil {
		if err.Error() != "sql: database is closed" {
			a.logger.Warn(fmt.Sprintf("Failed to persist last fetch time: %v", err))
		}
	}

	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
}

func (a *App) performInitialResolvedFetch() {
	if a.client == nil {
		return
	}

	// Use the resolved fetch mutex to prevent concurrent fetches
	a.resolvedFetchMu.Lock()
	defer a.resolvedFetchMu.Unlock()

	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Start with 3 days ago
	since := time.Now().Add(-72 * time.Hour)
	until := time.Now()

	a.logger.Info(fmt.Sprintf("Performing initial resolved incidents fetch from %s", since.Format(time.RFC3339)))

	// Update the latest resolved date
	a.latestResolvedMu.Lock()
	a.latestResolvedDate = since
	a.latestResolvedMu.Unlock()

	opts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      since,
		Until:      until,
	}

	// Use smaller page size for initial fetch
	incidents, err := a.client.FetchIncidentsWithPagination(opts, 50)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Initial resolved fetch failed: %v", err))
		return
	}

	// Update database
	updateCount := 0
	var latestDate time.Time
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert incident: %v", err))
		} else {
			updateCount++
			if incident.UpdatedAt.After(latestDate) {
				latestDate = incident.UpdatedAt
			}
		}
	}

	// Update latest resolved date
	if !latestDate.IsZero() {
		a.latestResolvedMu.Lock()
		a.latestResolvedDate = latestDate
		a.latestResolvedMu.Unlock()

		// Persist to database
		if err := a.db.SetState("latest_resolved_date", latestDate.Format(time.RFC3339)); err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to persist initial latest resolved date: %v", err))
		}
	}

	a.logger.Info(fmt.Sprintf("Initial fetch complete: fetched=%d, updated=%d resolved incidents", len(incidents), updateCount))
	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
}

func (a *App) fetchWithRetry(
	fn func() ([]database.IncidentData, error),
	maxRetries int, // parameter kept for compatibility but ignored
) ([]database.IncidentData, error) {
	// No retries - the polling mechanism handles automatic retries
	return fn()
}

func (a *App) refreshUserCache() {
	if a.client == nil {
		return
	}

	user, err := a.client.GetCurrentUser()
	if err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to refresh user cache: %v", err))
		return
	}

	a.userCache.Set(user.ID, user)
	a.logger.Debug("User cache refreshed successfully")
}

func (a *App) GetOpenIncidents(serviceIDs []string) ([]database.IncidentData, error) {
	if a.db == nil {
		err := fmt.Errorf("database not initialized")
		a.logger.Error(err.Error())
		return nil, err
	}

	// Filter out disabled services
	a.mu.RLock()
	enabledServices := []string{}
	if a.servicesConfig != nil {
		for _, serviceID := range serviceIDs {
			isDisabled := false
			for _, service := range a.servicesConfig.Services {
				// Check if this service is disabled
				if service.Disabled {
					switch id := service.ID.(type) {
					case string:
						if id == serviceID {
							isDisabled = true
							break
						}
					case []interface{}:
						for _, sid := range id {
							if strID, ok := sid.(string); ok && strID == serviceID {
								isDisabled = true
								break
							}
						}
					}
				}
				if isDisabled {
					break
				}
			}
			if !isDisabled {
				enabledServices = append(enabledServices, serviceID)
			}
		}
	} else {
		enabledServices = serviceIDs
	}
	a.mu.RUnlock()

	// Don't fetch if polling is active - just return cached data
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
	if len(enabledServices) == 0 {
		return allIncidents, nil
	}

	// Filter by enabled services only
	serviceMap := make(map[string]bool)
	for _, id := range enabledServices {
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

func (a *App) ToggleServiceDisabled(serviceID interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.servicesConfig == nil {
		return fmt.Errorf("no services configuration loaded")
	}

	// Find and toggle the service's disabled state
	for i := range a.servicesConfig.Services {
		service := &a.servicesConfig.Services[i]

		// Match service by ID
		match := false
		switch sid := service.ID.(type) {
		case string:
			if id, ok := serviceID.(string); ok && sid == id {
				match = true
			}
		case []interface{}:
			if idStr, ok := serviceID.(string); ok {
				for _, s := range sid {
					if str, ok := s.(string); ok && str == idStr {
						match = true
						break
					}
				}
			} else if idArr, ok := serviceID.([]interface{}); ok {
				// Compare arrays
				if len(sid) == len(idArr) {
					match = true
					for j, v := range sid {
						if str1, ok1 := v.(string); ok1 {
							if str2, ok2 := idArr[j].(string); ok2 {
								if str1 != str2 {
									match = false
									break
								}
							} else {
								match = false
								break
							}
						}
					}
				}
			}
		}

		if match {
			service.Disabled = !service.Disabled
			a.logger.Info(fmt.Sprintf("Service %s disabled state: %v", service.Name, service.Disabled))

			// Trigger immediate refresh
			go a.fetchAndUpdateIncidents()

			// Emit event to update UI
			runtime.EventsEmit(a.ctx, "services-config-updated")
			return nil
		}
	}

	return fmt.Errorf("service not found")
}

func (a *App) GetResolvedIncidents(
	serviceIDs []string) (
	[]database.IncidentData, error,
) {
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
		// Return cached data immediately WITHOUT spawning background fetch
		// The regular polling will keep data updated
		return cachedIncidents, nil
	}

	// No cache, fetch synchronously with proper mutex to prevent concurrent fetches
	a.resolvedFetchMu.Lock()
	defer a.resolvedFetchMu.Unlock()

	// Check again after acquiring lock (double-check pattern)
	cachedIncidents, err = a.db.GetResolvedIncidentsByServices(serviceIDs)
	if err == nil && len(cachedIncidents) > 0 {
		return cachedIncidents, nil
	}

	// Fetch from PagerDuty with proper timeout
	opts := store.FetchOptions{
		ServiceIDs: serviceIDs,
		Statuses:   []string{"resolved"},
		Since:      time.Now().Add(-48 * time.Hour),
	}

	incidents, err := a.client.FetchIncidentsWithPagination(opts, 50)
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

// GetIncidentSidebarData fetches alerts and notes for an incident with caching and deduplication
func (a *App) GetIncidentSidebarData(incidentID string) (*store.IncidentSidebarData, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("incident ID is required")
	}

	if a.client == nil {
		return nil, fmt.Errorf("PagerDuty client not initialized")
	}

	// Prevent duplicate fetches for the same incident - KEEP THIS LOGIC
	a.sidebarFetchingMu.Lock()
	if a.fetchingIncidents == nil {
		a.fetchingIncidents = make(map[string]bool)
	}

	if a.fetchingIncidents[incidentID] {
		a.sidebarFetchingMu.Unlock()
		// Check database for existing data
		dbAlerts, _ := a.db.GetIncidentAlerts(incidentID)
		dbNotes, _ := a.db.GetIncidentNotes(incidentID)

		if len(dbAlerts) > 0 || len(dbNotes) > 0 {
			// Convert database types to store types
			alerts := convertDBToStoreAlerts(dbAlerts)
			notes := convertDBToStoreNotes(dbNotes)

			return &store.IncidentSidebarData{
				IncidentID: incidentID,
				Alerts:     alerts,
				Notes:      notes,
				Loading:    false,
			}, nil
		}
		return nil, fmt.Errorf("fetch already in progress")
	}

	a.fetchingIncidents[incidentID] = true
	a.sidebarFetchingMu.Unlock()

	// Ensure we remove the fetching flag when done
	defer func() {
		a.sidebarFetchingMu.Lock()
		delete(a.fetchingIncidents, incidentID)
		a.sidebarFetchingMu.Unlock()
	}()

	// Create response object
	response := &store.IncidentSidebarData{
		IncidentID: incidentID,
		Loading:    false,
		Alerts:     []store.IncidentAlert{},
		Notes:      []store.IncidentNote{},
	}

	// Fetch from database first
	dbExistingAlerts, _ := a.db.GetIncidentAlerts(incidentID)
	dbExistingNotes, _ := a.db.GetIncidentNotes(incidentID)
	metadata, _ := a.db.GetSidebarMetadata(incidentID)

	// Convert database types to store types for existing data
	existingAlerts := convertDBToStoreAlerts(dbExistingAlerts)
	existingNotes := convertDBToStoreNotes(dbExistingNotes)

	// Get current incident data for comparison
	var currentIncident database.IncidentData
	incidents, err := a.db.GetOpenIncidents()
	if err == nil {
		for _, inc := range incidents {
			if inc.IncidentID == incidentID {
				currentIncident = inc
				break
			}
		}
	}

	// If no current incident found, check resolved
	if currentIncident.IncidentID == "" {
		resolved, err := a.db.GetResolvedIncidents()
		if err == nil {
			for _, inc := range resolved {
				if inc.IncidentID == incidentID {
					currentIncident = inc
					break
				}
			}
		}
	}

	// Decision logic for alerts
	shouldFetchAlerts := false
	if len(existingAlerts) == 0 {
		// First time fetching
		shouldFetchAlerts = true
	} else if metadata != nil {
		// Check if alert count changed
		if currentIncident.AlertCount != metadata.LastAlertCount {
			shouldFetchAlerts = true
		}
		// Check if last fetch was more than 3 minutes ago
		if metadata.LastFetchedAlerts != nil && time.Since(*metadata.LastFetchedAlerts) > 3*time.Minute && currentIncident.AlertCount > 0 {
			shouldFetchAlerts = true
		}
	}

	// Decision logic for notes
	shouldFetchNotes := false
	if len(existingNotes) == 0 {
		// First time fetching
		shouldFetchNotes = true
	} else if metadata != nil {
		// Check if incident was updated
		if metadata.LastUpdatedAt != nil && currentIncident.UpdatedAt.After(*metadata.LastUpdatedAt) {
			shouldFetchNotes = true
		}
		// Check if last fetch was more than 3 minutes ago
		if metadata.LastFetchedNotes != nil && time.Since(*metadata.LastFetchedNotes) > 3*time.Minute {
			shouldFetchNotes = true
		}
	}

	// Use existing data if no fetch needed
	if !shouldFetchAlerts && !shouldFetchNotes {
		response.Alerts = existingAlerts
		response.Notes = existingNotes
		return response, nil
	}

	// Concurrent API calls if needed
	type alertResult struct {
		alerts []store.IncidentAlert
		err    error
	}

	type noteResult struct {
		notes []store.IncidentNote
		err   error
	}

	alertChan := make(chan alertResult, 1)
	noteChan := make(chan noteResult, 1)

	// Fetch alerts if needed
	if shouldFetchAlerts {
		go func() {
			alerts, err := a.client.GetIncidentAlerts(incidentID)
			alertChan <- alertResult{alerts: alerts, err: err}
		}()
	} else {
		// Use existing alerts
		go func() {
			alertChan <- alertResult{alerts: existingAlerts, err: nil}
		}()
	}

	// Fetch notes if needed
	if shouldFetchNotes {
		go func() {
			notes, err := a.client.GetIncidentNotes(incidentID)
			noteChan <- noteResult{notes: notes, err: err}
		}()
	} else {
		// Use existing notes
		go func() {
			noteChan <- noteResult{notes: existingNotes, err: nil}
		}()
	}

	// Wait for both results with timeout
	timeout := time.After(30 * time.Second)
	var alertsReceived, notesReceived bool
	var errors []string
	var fetchedAlertsSuccess, fetchedNotesSuccess bool

	for !alertsReceived || !notesReceived {
		select {
		case alertRes := <-alertChan:
			alertsReceived = true
			if alertRes.err != nil {
				errors = append(errors, fmt.Sprintf("alerts: %v", alertRes.err))
				a.logger.Error(fmt.Sprintf("Failed to fetch alerts for %s: %v", incidentID, alertRes.err))
				// Use stale data on error
				response.Alerts = existingAlerts
			} else {
				response.Alerts = alertRes.alerts
				if shouldFetchAlerts {
					// Convert to database types and store
					dbAlerts := convertStoreToDBalerts(alertRes.alerts)
					if err := a.db.StoreIncidentAlerts(incidentID, dbAlerts); err != nil {
						a.logger.Error(fmt.Sprintf("Failed to store alerts: %v", err))
					} else {
						fetchedAlertsSuccess = true
					}
				}
			}

		case noteRes := <-noteChan:
			notesReceived = true
			if noteRes.err != nil {
				errors = append(errors, fmt.Sprintf("notes: %v", noteRes.err))
				a.logger.Error(fmt.Sprintf("Failed to fetch notes for %s: %v", incidentID, noteRes.err))
				// Use stale data on error
				response.Notes = existingNotes
			} else {
				response.Notes = noteRes.notes
				if shouldFetchNotes {
					// Convert to database types and store
					dbNotes := convertStoreToDbnotes(noteRes.notes)
					if err := a.db.StoreIncidentNotes(incidentID, dbNotes); err != nil {
						a.logger.Error(fmt.Sprintf("Failed to store notes: %v", err))
					} else {
						fetchedNotesSuccess = true
					}
				}
			}

		case <-timeout:
			if !alertsReceived || !notesReceived {
				errors = append(errors, "timeout waiting for data")
			}
			alertsReceived = true
			notesReceived = true
			// Use whatever existing data we have
			if !alertsReceived {
				response.Alerts = existingAlerts
			}
			if !notesReceived {
				response.Notes = existingNotes
			}
		}
	}

	// Update metadata if any successful fetches
	if (fetchedAlertsSuccess || fetchedNotesSuccess) && currentIncident.IncidentID != "" {
		err := a.db.UpdateSidebarMetadata(
			incidentID,
			currentIncident.AlertCount,
			currentIncident.UpdatedAt,
			fetchedAlertsSuccess,
			fetchedNotesSuccess,
		)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to update sidebar metadata: %v", err))
		}
	}

	// Set error if any occurred
	if len(errors) > 0 {
		response.Error = strings.Join(errors, "; ")
	}

	return response, nil
}

func convertDBToStoreAlerts(dbAlerts []database.SidebarAlert) []store.IncidentAlert {
	alerts := make([]store.IncidentAlert, len(dbAlerts))
	for i, dbAlert := range dbAlerts {
		alert := store.IncidentAlert{
			ID:          dbAlert.ID,
			Summary:     dbAlert.Summary,
			Status:      dbAlert.Status,
			CreatedAt:   dbAlert.CreatedAt,
			ServiceName: dbAlert.ServiceName,
			Links:       []store.AlertLink{},
		}

		// Deserialize links from JSON
		if dbAlert.Links != "" {
			json.Unmarshal([]byte(dbAlert.Links), &alert.Links)
		}

		alerts[i] = alert
	}
	return alerts
}

// Helper function to convert store alerts to database alerts
func convertStoreToDBalerts(storeAlerts []store.IncidentAlert) []database.SidebarAlert {
	dbAlerts := make([]database.SidebarAlert, len(storeAlerts))
	for i, storeAlert := range storeAlerts {
		// Serialize links to JSON
		linksJSON, _ := json.Marshal(storeAlert.Links)

		dbAlerts[i] = database.SidebarAlert{
			ID:          storeAlert.ID,
			Summary:     storeAlert.Summary,
			Status:      storeAlert.Status,
			CreatedAt:   storeAlert.CreatedAt,
			ServiceName: storeAlert.ServiceName,
			Links:       string(linksJSON),
		}
	}
	return dbAlerts
}

// Helper function to convert database notes to store notes
func convertDBToStoreNotes(dbNotes []database.SidebarNote) []store.IncidentNote {
	notes := make([]store.IncidentNote, len(dbNotes))
	for i, dbNote := range dbNotes {
		note := store.IncidentNote{
			ID:              dbNote.ID,
			Content:         dbNote.Content,
			CreatedAt:       dbNote.CreatedAt,
			UserName:        dbNote.UserName,
			ServiceID:       dbNote.ServiceID,
			FreeformContent: dbNote.FreeformContent,
		}

		// Deserialize responses from JSON
		if dbNote.Responses != "" {
			var responses []store.NoteResponse
			if err := json.Unmarshal([]byte(dbNote.Responses), &responses); err == nil {
				note.Responses = responses
			}
		}

		// Deserialize tags from JSON
		if dbNote.Tags != "" {
			var tags []store.NoteTag
			if err := json.Unmarshal([]byte(dbNote.Tags), &tags); err == nil {
				note.Tags = tags
			}
		}

		notes[i] = note
	}
	return notes
}

// Helper function to convert store notes to database notes
func convertStoreToDbnotes(storeNotes []store.IncidentNote) []database.SidebarNote {
	dbNotes := make([]database.SidebarNote, len(storeNotes))
	for i, storeNote := range storeNotes {
		dbNote := database.SidebarNote{
			ID:              storeNote.ID,
			Content:         storeNote.Content,
			CreatedAt:       storeNote.CreatedAt,
			UserName:        storeNote.UserName,
			ServiceID:       storeNote.ServiceID,
			FreeformContent: storeNote.FreeformContent,
		}

		// Serialize responses to JSON
		if len(storeNote.Responses) > 0 {
			if responsesJSON, err := json.Marshal(storeNote.Responses); err == nil {
				dbNote.Responses = string(responsesJSON)
			}
		}

		// Serialize tags to JSON
		if len(storeNote.Tags) > 0 {
			if tagsJSON, err := json.Marshal(storeNote.Tags); err == nil {
				dbNote.Tags = string(tagsJSON)
			}
		}

		dbNotes[i] = dbNote
	}
	return dbNotes
}

func (a *App) GetServiceConfigByServiceID(serviceID string) (*store.ServiceConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.servicesConfig == nil {
		return nil, fmt.Errorf("no services configuration loaded")
	}

	for _, service := range a.servicesConfig.Services {
		switch id := service.ID.(type) {
		case string:
			if id == serviceID {
				return &service, nil
			}
		case []interface{}:
			for _, sid := range id {
				if strID, ok := sid.(string); ok && strID == serviceID {
					return &service, nil
				}
			}
		case float64:
			if fmt.Sprintf("%.0f", id) == serviceID {
				return &service, nil
			}
		}
	}

	return nil, fmt.Errorf("service not found: %s", serviceID)
}

func (a *App) cleanupOldSidebarData() {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	a.shutdownWg.Add(1)
	defer a.shutdownWg.Done()

	for {
		select {
		case <-a.shutdownChan:
			a.logger.Info("Sidebar cleanup routine stopped by shutdown signal")
			return
		case <-ticker.C:
			cutoffDate := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago

			if err := a.db.CleanupOldSidebarData(cutoffDate); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to cleanup old sidebar data: %v", err))
			} else {
				a.logger.Info("Successfully cleaned up old sidebar data")
			}
		}
	}
}

// to fetch user on startup
func (a *App) ConfigureAPIKey(
	apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Initialize PagerDuty client with queue
	client, err := store.NewClient(apiKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to create PagerDuty client: %v", err))
		return fmt.Errorf("failed to create PagerDuty client: %w", err)
	}

	// Set custom logger for the client
	if a.logger != nil {
		client.SetLogger(func(msg string) {
			a.logger.Info(msg)
		})
	}

	// Test the API key by getting current user and cache the user ID
	user, err := client.GetCurrentUser()
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
		}
	}

	// Update client
	a.client = client
	a.logger.Info("API key configured successfully")

	// Initialize components if not already done
	if a.userCache == nil {
		a.userCache = NewUserCache()
	}
	if a.circuitBreaker == nil {
		a.circuitBreaker = NewCircuitBreaker()
	}
	if a.rateLimitTracker == nil {
		a.rateLimitTracker = NewRateLimitTracker()
	}

	// Cache the user ID immediately
	a.userCache.Set(user.ID, user)
	a.logger.Info(fmt.Sprintf("Cached user ID: %s", user.ID))

	// Start polling cycles
	a.StartPolling()
	a.StartUserPolling()
	a.StartResolvedPolling()

	// Emit event to notify UI
	runtime.EventsEmit(a.ctx, "api-key-configured")

	return nil
}

func (a *App) GetAPIKey() (
	string, error,
) {
	if a.kr == nil {
		return "", fmt.Errorf("keyring not available")
	}

	item, err := a.kr.Get("pagerduty-api-key")
	if err != nil {
		return "", err
	}

	return string(item.Data), nil
}

func (a *App) UploadServicesConfig(
	jsonData string) error {
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

	// Invalidate user cache on service change
	if a.userCache != nil {
		a.userCache.Invalidate()
	}

	// Trigger immediate refresh
	go a.fetchAndUpdateIncidents()
	go a.fetchResolvedIncidentsAdaptive()

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "services-config-updated")

	return nil
}

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

func (a *App) GetServicesConfig() (
	*store.ServicesConfig, error,
) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.servicesConfig == nil {
		return nil, fmt.Errorf("no services configuration loaded")
	}
	return a.servicesConfig, nil
}

func (a *App) GetServiceNameByID(serviceID string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.servicesConfig == nil {
		return ""
	}

	for _, service := range a.servicesConfig.Services {
		switch id := service.ID.(type) {
		case string:
			if id == serviceID {
				return service.Name
			}
		case []interface{}:
			for _, sid := range id {
				if strID, ok := sid.(string); ok && strID == serviceID {
					return service.Name
				}
			}
		case float64:
			if fmt.Sprintf("%.0f", id) == serviceID {
				return service.Name
			}
		}
	}

	return ""
}

func (a *App) SetSelectedServices(
	services []string,
) {
	a.mu.Lock()
	oldServices := a.selectedServices
	a.selectedServices = services
	a.mu.Unlock()

	// Check if services actually changed
	if !slicesEqual(oldServices, services) {
		a.logger.Debug(fmt.Sprintf("Selected services updated: %d services", len(services)))

		// Invalidate user cache on service change
		if a.userCache != nil {
			a.userCache.Invalidate()
		}

		// Trigger immediate refresh for both open and resolved
		go a.fetchAndUpdateIncidents()
		go a.fetchResolvedIncidentsAdaptive()
	}
}

func (a *App) GetSelectedServices() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]string{}, a.selectedServices...)
}

func (a *App) ReadFile(
	path string) (
	string, error,
) {
	content, err := os.ReadFile(path)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to read file %s: %v", path, err))
		return "", err
	}
	return string(content), nil
}

func (a *App) GetRateLimitStatus() map[string]interface{} {
	currentRate := a.rateLimitTracker.GetCurrentRate()
	status := map[string]interface{}{
		"current":    currentRate,
		"max":        960,
		"remaining":  960 - currentRate,
		"percentage": float64(currentRate) / 960.0 * 100,
	}

	if a.circuitBreaker != nil {
		status["circuit_breaker"] = map[string]interface{}{
			"state":    atomic.LoadInt32(&a.circuitBreaker.state),
			"failures": atomic.LoadInt32(&a.circuitBreaker.failures),
		}
	}

	return status
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

func (a *App) TestNotificationSound() error {
	if a.notificationMgr != nil {
		return a.notificationMgr.TestSound()
	}
	return fmt.Errorf("notification manager not initialized")
}

func (a *App) GetAvailableSounds() []string {
	if a.notificationMgr != nil {
		sounds, err := a.notificationMgr.GetAvailableSounds()
		if err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to get available sounds: %v", err))
			return []string{"default"}
		}
		return sounds
	}
	return []string{"default"}
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

func (a *App) IsNotificationSnoozed() bool {
	if a.notificationMgr != nil {
		return a.notificationMgr.IsSnoozeActive()
	}
	return false
}

func (a *App) shutdown(ctx context.Context) {
	a.logger.Info("Shutting down PagerOps...")

	// First, stop all polling to prevent new operations
	a.StopPolling()
	a.StopUserPolling()
	a.StopResolvedPolling()

	// Then signal shutdown to running goroutines
	close(a.shutdownChan)

	// Shutdown notification manager
	if a.notificationMgr != nil {
		a.notificationMgr.Shutdown()
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		a.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All goroutines stopped")
	case <-time.After(10 * time.Second):
		a.logger.Warn("Shutdown timeout - some goroutines may not have stopped")
	}

	// Shutdown the client API queue
	if a.client != nil {
		a.client.Shutdown()
	}

	// Close database
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to close database: %v", err))
		}
	}

	// Close logger
	if a.logger != nil {
		a.logger.Close()
	}
}

// Helper function to compare slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// NoteInput represents the structured note data from the frontend
type NoteInput struct {
	Responses      []store.NoteResponse `json:"responses"`
	Tags           []store.NoteTag      `json:"tags"`
	FreeformContent string               `json:"freeform_content"`
}

// getUserEmail retrieves the current user's email from cache
func (a *App) getUserEmail() (string, error) {
	if a.userCache == nil {
		return "", fmt.Errorf("user cache not initialized")
	}

	// Try to get from cache first
	a.userCache.mu.RLock()
	user := a.userCache.user
	a.userCache.mu.RUnlock()

	if user != nil {
		if pdUser, ok := user.(*pagerduty.User); ok && pdUser.Email != "" {
			return pdUser.Email, nil
		}
	}

	// If not in cache or no email, fetch fresh user data
	if a.client == nil {
		return "", fmt.Errorf("PagerDuty client not initialized")
	}

	freshUser, err := a.client.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	// Update cache with fresh data
	a.userCache.Set(freshUser.ID, freshUser)

	if freshUser.Email == "" {
		return "", fmt.Errorf("user email not available")
	}

	return freshUser.Email, nil
}

// AcknowledgeIncident acknowledges an incident via the PagerDuty API
func (a *App) AcknowledgeIncident(incidentID string) error {
	if incidentID == "" {
		return fmt.Errorf("incident ID is required")
	}

	if a.client == nil {
		return fmt.Errorf("PagerDuty client not initialized")
	}

	// Get current user's email
	userEmail, err := a.getUserEmail()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to get user email for acknowledge: %v", err))
		return fmt.Errorf("failed to get user email: %w", err)
	}

	a.logger.Info(fmt.Sprintf("Acknowledging incident %s as user %s", incidentID, userEmail))

	// Call API to acknowledge incident
	err = a.client.AcknowledgeIncident(incidentID, userEmail)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to acknowledge incident %s: %v", incidentID, err))
		return fmt.Errorf("failed to acknowledge incident: %w", err)
	}

	a.logger.Info(fmt.Sprintf("Successfully acknowledged incident %s", incidentID))

	// Trigger immediate fetch to update UI quickly
	// The polling will also pick this up, but this provides instant feedback
	go a.fetchAndUpdateIncidents()

	return nil
}

// AddIncidentNote adds a note to an incident via the PagerDuty API
func (a *App) AddIncidentNote(incidentID string, noteData NoteInput) error {
	if incidentID == "" {
		return fmt.Errorf("incident ID is required")
	}

	if a.client == nil {
		return fmt.Errorf("PagerDuty client not initialized")
	}

	// Format the note content from structured data
	formattedContent := store.FormatNoteContent(noteData.Responses, noteData.Tags, noteData.FreeformContent)

	// Validate that there is content
	if strings.TrimSpace(formattedContent) == "" {
		return fmt.Errorf("note cannot be empty")
	}

	a.logger.Info(fmt.Sprintf("Adding note to incident %s", incidentID))

	// Call API to create the note
	err := a.client.CreateIncidentNote(incidentID, formattedContent)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to add note to incident %s: %v", incidentID, err))
		return fmt.Errorf("failed to add note: %w", err)
	}

	a.logger.Info(fmt.Sprintf("Successfully added note to incident %s", incidentID))

	// Clear sidebar cache for this incident to force refetch
	// This ensures the new note appears immediately
	if clearErr := a.db.ClearIncidentSidebarCache(incidentID); clearErr != nil {
		a.logger.Warn(fmt.Sprintf("Failed to clear sidebar cache: %v", clearErr))
		// Don't fail the operation if cache clear fails
	}

	// Emit event to refresh sidebar
	runtime.EventsEmit(a.ctx, "sidebar-data-updated", incidentID)

	return nil
}
