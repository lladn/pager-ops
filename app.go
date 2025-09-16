package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"pager-ops/database"
	"pager-ops/store"

	"github.com/99designs/keyring"
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
	// NEW FIELDS for user polling
	userPolling    bool
	userPollTicker *time.Ticker
	userPollMu     sync.RWMutex
	// NEW FIELDS for resolved date tracking
	latestResolvedDate time.Time
	latestResolvedMu   sync.RWMutex
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
	failures       int32
	lastFailure    time.Time
	state          int32 // 0: closed, 1: open, 2: half-open
	maxFailures    int32
	cooldownPeriod time.Duration
	mu             sync.RWMutex
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
		maxFailures:    5,
		cooldownPeriod: 30 * time.Second,
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
		cb.mu.RUnlock()

		if time.Since(lastFailure) > cb.cooldownPeriod {
			atomic.StoreInt32(&cb.state, 2) // Half-open
			return true
		}
		return false
	}

	return true // Half-open
}

func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, 0) // Closed
}

func (cb *CircuitBreaker) RecordFailure() {
	failures := atomic.AddInt32(&cb.failures, 1)

	cb.mu.Lock()
	cb.lastFailure = time.Now()
	cb.mu.Unlock()

	if failures >= cb.maxFailures {
		atomic.StoreInt32(&cb.state, 1) // Open
	}
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
	}
}

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

	// Initialize production components
	a.rateLimitTracker = NewRateLimitTracker()
	a.userCache = NewUserCache()
	a.circuitBreaker = NewCircuitBreaker()

	// Try to load API key and initialize client
	apiKey, err := a.GetAPIKey()
	if err == nil && apiKey != "" {
		client, err := store.NewClient(apiKey)
		if err == nil {
			a.client = client
			a.logger.Info("PagerDuty client initialized successfully")

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

func (a *App) processAndUpdateIncidents(incidents []database.IncidentData, source string) {
	// Check shutdown before database operations
	select {
	case <-a.shutdownChan:
		return
	default:
	}

	// Update database with new incidents from this source
	updatedCount := 0
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			if err.Error() == "sql: database is closed" {
				a.logger.Info("Database closed, stopping incident updates")
				return
			}
			a.logger.Error(fmt.Sprintf("Failed to upsert incident: %v", err))
		} else {
			updatedCount++
		}
	}

	if updatedCount > 0 {
		a.logger.Debug(fmt.Sprintf("[%s] Updated %d incidents in database", source, updatedCount))
	}

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

	// Get previous state
	a.previousOpenMu.RLock()
	previousOpen := make(map[string]database.IncidentData)
	for k, v := range a.previousOpenIncidents {
		previousOpen[k] = v
	}
	a.previousOpenMu.RUnlock()

	// Detect REAL status transitions (only if incident existed before and now doesn't)
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
		a.shutdownWg.Add(1)
		go func() {
			defer a.shutdownWg.Done()
			a.fetchRecentResolvedIncidents()
		}()
	}

	a.previousOpenMu.Lock()
	a.previousOpenIncidents = currentOpen
	a.previousOpenMu.Unlock()

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "incidents-updated", "both")

	// Check for triggered incidents and send notifications
	a.checkForTriggeredIncidents()
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


func (a *App) StartPolling() {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	if a.polling {
		return
	}

	a.polling = true
	a.pollTicker = time.NewTicker(3 * time.Second)
	a.logger.Info("Started service incidents polling (3s interval)")

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
			case <-a.pollTicker.C:
				// Check polling state with lock
				a.pollMu.RLock()
				shouldContinue := a.polling
				a.pollMu.RUnlock()

				if !shouldContinue {
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
			case <-a.userPollTicker.C:
				// Check polling state with lock
				a.userPollMu.RLock()
				shouldContinue := a.userPolling
				a.userPollMu.RUnlock()

				if !shouldContinue {
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
			case <-a.resolvedPollTicker.C:
				a.resolvedPollMu.RLock()
				shouldContinue := a.resolvedPolling
				a.resolvedPollMu.RUnlock()

				if !shouldContinue {
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

	// Get selected services
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
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

	// Fetch incidents assigned to user
	incidents, err := a.fetchWithRetry(func() ([]database.IncidentData, error) {
		return a.client.FetchOpenIncidents([]string{}, userID)
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

	// Get the latest resolved date with read lock
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

	a.logger.Info(fmt.Sprintf("Fetching resolved incidents since: %s", since.Format(time.RFC3339)))

	resolvedOpts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      since,
		Until:      now,
	}

	// Use paginated fetch for resolved incidents
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

	// Update database and track latest date
	var latestDate time.Time
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			if err.Error() == "sql: database is closed" {
				a.logger.Info("Database closed, stopping resolved incident updates")
				return
			}
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}

		// Track the latest resolved date
		if incident.UpdatedAt.After(latestDate) {
			latestDate = incident.UpdatedAt
		}
	}

	// Update latest resolved date if we found newer incidents
	if !latestDate.IsZero() && latestDate.After(since) {
		a.latestResolvedMu.Lock()
		a.latestResolvedDate = latestDate
		a.latestResolvedMu.Unlock()

		// Persist to database
		if err := a.db.SetState("latest_resolved_date", latestDate.Format(time.RFC3339)); err != nil {
			if err.Error() != "sql: database is closed" {
				a.logger.Warn(fmt.Sprintf("Failed to persist latest resolved date: %v", err))
			}
		}
		a.logger.Info(fmt.Sprintf("Updated latest resolved date to: %s", latestDate.Format(time.RFC3339)))
	}

	if len(incidents) > 0 {
		runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
		a.logger.Info(fmt.Sprintf("Processed %d resolved incidents", len(incidents)))
	}
}


// fetchResolvedIncidents - Original method preserved for compatibility
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

	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Query database for the newest resolved incident date
	var since time.Time
	newestResolved, err := a.db.GetNewestResolvedIncidentDate()
	if err == nil && !newestResolved.IsZero() {
		// Use the newest date from database, or 3 days ago, whichever is more recent
		threeDaysAgo := time.Now().Add(-72 * time.Hour)
		if newestResolved.After(threeDaysAgo) {
			since = newestResolved
			a.logger.Info(fmt.Sprintf("Using database newest resolved date: %s", since.Format(time.RFC3339)))
		} else {
			since = threeDaysAgo
			a.logger.Info("Database date too old, using 3 days ago")
		}
	} else {
		// No resolved incidents in database, fetch last 3 days
		since = time.Now().Add(-72 * time.Hour)
		a.logger.Info("No resolved incidents in database, fetching last 3 days")
	}

	// Update the latest resolved date
	a.latestResolvedMu.Lock()
	a.latestResolvedDate = since
	a.latestResolvedMu.Unlock()

	until := time.Now()
	a.logger.Info(fmt.Sprintf("Performing initial resolved incidents fetch from %s", since.Format(time.RFC3339)))

	opts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      since,
		Until:      until,
	}

	// Use paginated fetch
	incidents, err := a.client.FetchIncidentsWithPagination(opts, 100)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Initial resolved fetch failed: %v", err))
		return
	}

	// Batch update database
	if err := a.db.BatchUpsertIncidents(incidents); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to batch upsert incidents: %v", err))
	}

	// Update latest resolved date
	var latestDate time.Time
	for _, incident := range incidents {
		if incident.UpdatedAt.After(latestDate) {
			latestDate = incident.UpdatedAt
		}
	}

	if !latestDate.IsZero() {
		a.latestResolvedMu.Lock()
		a.latestResolvedDate = latestDate
		a.latestResolvedMu.Unlock()

		// Persist to database
		if err := a.db.SetState("latest_resolved_date", latestDate.Format(time.RFC3339)); err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to persist initial latest resolved date: %v", err))
		}
	}

	a.logger.Info(fmt.Sprintf("Initial fetch complete: %d resolved incidents", len(incidents)))
	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
}


func (a *App) fetchRecentResolvedIncidents() {
	if a.client == nil || !a.rateLimitTracker.CanMakeCall() {
		return
	}

	// Prevent duplicate fetches within 5 seconds
	a.lastResolvedFetchMu.RLock()
	lastFetch := a.lastResolvedFetch
	a.lastResolvedFetchMu.RUnlock()
	
	if time.Since(lastFetch) < 5*time.Second {
		a.logger.Debug("Skipping recent resolved fetch - too soon since last fetch")
		return
	}

	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	if len(selectedServices) == 0 {
		return
	}

	// Update last fetch time
	a.lastResolvedFetchMu.Lock()
	a.lastResolvedFetch = time.Now()
	a.lastResolvedFetchMu.Unlock()

	// Fetch resolved incidents from last hour for immediate updates
	opts := store.FetchOptions{
		ServiceIDs: selectedServices,
		Statuses:   []string{"resolved"},
		Since:      time.Now().Add(-1 * time.Hour),
	}

	incidents, err := a.client.FetchIncidentsWithOptions(opts)
	if err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to fetch recent resolved: %v", err))
		return
	}

	// Update database and track latest date
	var latestDate time.Time
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}
		if incident.UpdatedAt.After(latestDate) {
			latestDate = incident.UpdatedAt
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

	a.rateLimitTracker.RecordCall()
	runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
	a.logger.Info(fmt.Sprintf("Immediate resolved update: %d incidents", len(incidents)))
}

func (a *App) fetchWithRetry(
	fn func() ([]database.IncidentData, error),
	maxRetries int,
) ([]database.IncidentData, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// Exponential backoff: 2^i seconds
			backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
			a.logger.Info(fmt.Sprintf("Retry %d/%d after %v", i, maxRetries, backoff))
			time.Sleep(backoff)
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		a.logger.Warn(fmt.Sprintf("Attempt %d failed: %v", i+1, err))
	}

	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
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
		a.shutdownWg.Add(1)
		go func() {
			defer a.shutdownWg.Done()
			
			// Check shutdown
			select {
			case <-a.shutdownChan:
				return
			default:
			}
			
			// Check rate limit
			if !a.rateLimitTracker.CanMakeCall() {
				return
			}

			// Fetch fresh data in background WITH PAGINATION for resolved
			opts := store.FetchOptions{
				ServiceIDs: serviceIDs,
				Statuses:   []string{"resolved"},
				Since:      time.Now().Add(-48 * time.Hour),
			}
			
			incidents, err := a.client.FetchIncidentsWithPagination(opts, 100)
			if err != nil {
				a.logger.Error(fmt.Sprintf("Failed to fetch resolved incidents: %v", err))
				return
			}
			
			// Check shutdown before database operations
			select {
			case <-a.shutdownChan:
				return
			default:
			}
			
			// Update database
			for _, incident := range incidents {
				if err := a.db.UpsertIncident(incident); err != nil {
					if err.Error() != "sql: database is closed" {
						a.logger.Error(fmt.Sprintf("Failed to upsert resolved incident: %v", err))
					}
				}
			}

			a.rateLimitTracker.RecordCall()
			// Emit event to update UI
			runtime.EventsEmit(a.ctx, "incidents-updated", "resolved")
		}()
		return cachedIncidents, nil
	}

	// No cache, fetch from PagerDuty WITH PAGINATION
	opts := store.FetchOptions{
		ServiceIDs: serviceIDs,
		Statuses:   []string{"resolved"},
		Since:      time.Now().Add(-48 * time.Hour),
	}
	
	incidents, err := a.client.FetchIncidentsWithPagination(opts, 100)
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


// ConfigureAPIKey - Original method unchanged
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

	// Initialize new components if not already done
	if a.userCache == nil {
		a.userCache = NewUserCache()
	}
	if a.circuitBreaker == nil {
		a.circuitBreaker = NewCircuitBreaker()
	}

	// Start all three polling cycles
	a.StartPolling()
	a.StartUserPolling()
	a.StartResolvedPolling()

	// Emit event to notify UI
	runtime.EventsEmit(a.ctx, "api-key-configured")

	return nil
}

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


func (a *App) GetServicesConfig() (*store.ServicesConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.servicesConfig == nil {
		return nil, fmt.Errorf("no services configuration loaded")
	}
	return a.servicesConfig, nil
}


func (a *App) SetSelectedServices(services []string) {
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


func (a *App) ReadFile(path string) (string, error) {
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
	a.logger.Info("PagerOps shutting down...")
	
	// Signal all goroutines to stop
	close(a.shutdownChan)
	
	// Stop all polling cycles
	a.StopPolling()
	a.StopUserPolling()
	a.StopResolvedPolling()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		a.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All goroutines stopped successfully")
	case <-time.After(5 * time.Second):
		a.logger.Warn("Timeout waiting for goroutines to stop")
	}

	// Close database after all goroutines have stopped
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to close database: %v", err))
		} else {
			a.logger.Info("Database closed successfully")
		}
	}

	// Close logger last
	if a.logger != nil {
		a.logger.Info("Shutdown complete")
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
