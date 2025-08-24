package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
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
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize database
	dbPath := filepath.Join(".", "incidents.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		runtime.LogError(ctx, fmt.Sprintf("Failed to initialize database: %v", err))
		return
	}
	a.db = db

	// Initialize keyring
	kr, err := keyring.Open(keyring.Config{
		ServiceName: "PagerOps",
	})
	if err != nil {
		runtime.LogWarning(ctx, fmt.Sprintf("Failed to initialize keyring: %v", err))
	}
	a.kr = kr

	// Try to load API key and initialize client
	apiKey, err := a.GetAPIKey()
	if err == nil && apiKey != "" {
		client, err := store.NewClient(apiKey)
		if err == nil {
			a.client = client
			// Start polling
			a.StartPolling()
		}
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.polling {
		a.StopPolling()
	}
	if a.db != nil {
		a.db.Close()
	}
}

// ConfigureAPIKey stores the API key in the keychain
func (a *App) ConfigureAPIKey(apiKey string) error {
	if a.kr == nil {
		return fmt.Errorf("keyring not initialized")
	}

	err := a.kr.Set(keyring.Item{
		Key:   "pagerduty_api_key",
		Data:  []byte(apiKey),
		Label: "PagerOps API key",
	})

	if err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Initialize client with new API key
	client, err := store.NewClient(apiKey)
	if err != nil {
		return fmt.Errorf("failed to initialize PagerDuty client: %w", err)
	}
	a.client = client

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
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	a.servicesConfig = &config

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
		}
	}

	return nil
}

// GetServicesConfig returns the current services configuration
func (a *App) GetServicesConfig() (*store.ServicesConfig, error) {
	if a.servicesConfig == nil {
		return nil, fmt.Errorf("no services configuration loaded")
	}
	return a.servicesConfig, nil
}

// SetSelectedServices updates the selected services for filtering
func (a *App) SetSelectedServices(services []string) {
	a.selectedServices = services
}

// StartPolling starts the incident polling
func (a *App) StartPolling() {
	if a.polling {
		return
	}

	a.polling = true
	a.pollTicker = time.NewTicker(3 * time.Second)

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
}

// fetchAndUpdateIncidents fetches incidents from PagerDuty and updates the database
func (a *App) fetchAndUpdateIncidents() {
	if a.client == nil {
		return
	}

	// Get current user
	user, err := a.client.GetCurrentUser()
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("Failed to get current user: %v", err))
		return
	}

	// Fetch open incidents
	incidents, err := a.client.FetchOpenIncidents(a.selectedServices, user.ID)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("Failed to fetch open incidents: %v", err))
		return
	}

	// Update database
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("Failed to upsert incident: %v", err))
		}
	}

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "incidents-updated", "open")
}

// GetOpenIncidents returns all open incidents from the database
func (a *App) GetOpenIncidents() ([]database.IncidentData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.db.GetOpenIncidents()
}

// GetResolvedIncidents fetches and returns resolved incidents
func (a *App) GetResolvedIncidents() ([]database.IncidentData, error) {
	if a.client == nil {
		return nil, fmt.Errorf("PagerDuty client not initialized")
	}

	// Fetch resolved incidents from PagerDuty
	incidents, err := a.client.FetchResolvedIncidents(a.selectedServices)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resolved incidents: %w", err)
	}

	// Update database
	for _, incident := range incidents {
		if err := a.db.UpsertIncident(incident); err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("Failed to upsert resolved incident: %v", err))
		}
	}

	// Return from database
	return a.db.GetResolvedIncidents()
}

// ReadFile reads a file and returns its content
func (a *App) ReadFile(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
