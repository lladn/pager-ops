package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pager-ops/database"
	"pager-ops/store"

	"github.com/99designs/keyring"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	db               *database.DB
	client           *store.Client
	polling          bool
	pollTicker       *time.Ticker
	servicesConfig   *store.ServicesConfig
	selectedServices []string
	kr               keyring.Keyring
	logger           *Logger
	mu               sync.RWMutex // Protect concurrent access
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
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

	// Try to load API key and initialize client
	apiKey, err := a.GetAPIKey()
	if err == nil && apiKey != "" {
		client, err := store.NewClient(apiKey)
		if err == nil {
			a.client = client
			a.logger.Info("PagerDuty client initialized successfully")
			// Start polling
			a.StartPolling()
		} else {
			a.logger.Error(fmt.Sprintf("Failed to initialize PagerDuty client: %v", err))
		}
	} else {
		a.logger.Info("No API key configured yet")
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.logger.Info("PagerOps shutting down...")
	
	if a.polling {
		a.StopPolling()
	}
	if a.db != nil {
		a.db.Close()
	}
	if a.logger != nil {
		a.logger.Close()
	}
}

// ConfigureAPIKey stores the API key in the keychain
func (a *App) ConfigureAPIKey(apiKey string) error {
	if a.kr == nil {
		err := fmt.Errorf("keyring not initialized")
		a.logger.Error(err.Error())
		return err
	}

	err := a.kr.Set(keyring.Item{
		Key:   "pagerduty_api_key",
		Data:  []byte(apiKey),
		Label: "PagerOps API key",
	})

	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to store API key: %v", err))
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Initialize client with new API key
	client, err := store.NewClient(apiKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to initialize PagerDuty client: %v", err))
		return fmt.Errorf("failed to initialize PagerDuty client: %w", err)
	}
	a.client = client
	a.logger.Info("API key configured successfully")

	// Start polling if not already running
	if !a.polling {
		a.StartPolling()
	}

	return nil
}

// GetAPIKey retrieves the API key from the keychain
func (a *App) GetAPIKey() (string, error) {
	if a.kr == nil {
		return "", fmt.Errorf("keyring not initialized")
	}

	item, err := a.kr.Get("pagerduty_api_key")
	if err != nil {
		return "", err
	}

	return string(item.Data), nil
}

// UploadServicesConfig processes the uploaded services configuration
func (a *App) UploadServicesConfig(content string) error {
	var config store.ServicesConfig
	err := json.Unmarshal([]byte(content), &config)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Invalid JSON format: %v", err))
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Lock for writing
	a.mu.Lock()
	defer a.mu.Unlock()

	a.servicesConfig = &config
	a.logger.Info(fmt.Sprintf("Services configuration uploaded with %d services", len(config.Services)))

	// Initialize selected services with all services
	a.selectedServices = []string{}
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

// StartPolling starts the incident polling
func (a *App) StartPolling() {
	if a.polling {
		return
	}

	a.polling = true
	a.pollTicker = time.NewTicker(30 * time.Second)
	a.logger.Info("Started incident polling (30s interval)")

	go func() {
		// Initial fetch
		a.fetchAndUpdateIncidents()

		for range a.pollTicker.C {
			if !a.polling {
				break
			}
			a.fetchAndUpdateIncidents()
		}
	}()
}

// StopPolling stops the incident polling
func (a *App) StopPolling() {
	a.polling = false
	if a.pollTicker != nil {
		a.pollTicker.Stop()
	}
	a.logger.Info("Stopped incident polling")
}

// fetchAndUpdateIncidents fetches incidents from PagerDuty and updates the database
func (a *App) fetchAndUpdateIncidents() {
	if a.client == nil {
		return
	}

	// Get selected services with read lock
	a.mu.RLock()
	selectedServices := append([]string{}, a.selectedServices...)
	a.mu.RUnlock()

	// Get current user
	user, err := a.client.GetCurrentUser()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to get current user: %v", err))
		return
	}

	// Fetch open incidents
	incidents, err := a.client.FetchOpenIncidents(selectedServices, user.ID)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch open incidents: %v", err))
		return
	}

	// Update database
	newIncidents := 0
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to upsert incident: %v", err))
		} else {
			newIncidents++
		}
	}

	if newIncidents > 0 {
		a.logger.Debug(fmt.Sprintf("Updated %d incidents in database", newIncidents))
	}

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "incidents-updated", "open")
}

// GetOpenIncidents returns open incidents filtered by selected services
func (a *App) GetOpenIncidents(serviceIDs []string) ([]database.IncidentData, error) {
	if a.db == nil {
		err := fmt.Errorf("database not initialized")
		a.logger.Error(err.Error())
		return nil, err
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

	// Get configured services if no specific services requested
	if len(serviceIDs) == 0 {
		a.mu.RLock()
		serviceIDs = append([]string{}, a.selectedServices...)
		a.mu.RUnlock()
	}

	// Only fetch if we have services configured
	if len(serviceIDs) == 0 {
		a.logger.Info("No services configured, returning empty resolved incidents")
		return []database.IncidentData{}, nil
	}

	// Fetch resolved incidents from PagerDuty
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

	// Return filtered incidents from database
	allResolved, err := a.db.GetResolvedIncidents()
	if err != nil {
		return nil, err
	}

	// Filter by service IDs
	serviceMap := make(map[string]bool)
	for _, id := range serviceIDs {
		serviceMap[id] = true
	}

	var filteredIncidents []database.IncidentData
	for _, incident := range allResolved {
		if serviceMap[incident.ServiceID] {
			filteredIncidents = append(filteredIncidents, incident)
		}
	}

	return filteredIncidents, nil
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