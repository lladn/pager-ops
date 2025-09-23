package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type NotificationConfig struct {
	Enabled     bool      `json:"enabled"`
	Sound       string    `json:"sound"`
	Snoozed     bool      `json:"snoozed"`
	SnoozeUntil time.Time `json:"snoozeUntil"`
}

// SoundRequest represents a sound playback request
type SoundRequest struct {
	Type        string // "default" or "custom"
	SoundFile   string // file for custom
	ServiceName string // service name for default say command
	ResultChan  chan error
}

// NotificationManager manages notifications and sounds
type NotificationManager struct {
	config      NotificationConfig
	mu          sync.RWMutex
	logger      *Logger
	soundQueue  chan SoundRequest
	rateLimiter *RateLimiter
	shutdownCh  chan struct{}
	wg          sync.WaitGroup
}

// RateLimiter implements a simple rate limiting mechanism
type RateLimiter struct {
	mu           sync.Mutex
	lastNotif    time.Time
	minInterval  time.Duration
	burstCount   int
	burstWindow  time.Duration
	windowStart  time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		minInterval: 2 * time.Second,        // Minimum 2 seconds between notifications
		burstWindow: 30 * time.Second,       // Window for burst detection
		burstCount:  0,
		windowStart: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	// Reset burst window if expired
	if now.Sub(rl.windowStart) > rl.burstWindow {
		rl.burstCount = 0
		rl.windowStart = now
	}

	// Check minimum interval
	if now.Sub(rl.lastNotif) < rl.minInterval {
		return false
	}

	// Check burst limit (max 5 notifications in 30 seconds)
	if rl.burstCount >= 5 {
		return false
	}

	rl.lastNotif = now
	rl.burstCount++
	return true
}

func NewNotificationManager(logger *Logger) *NotificationManager {
	nm := &NotificationManager{
		config: NotificationConfig{
			Enabled: true,
			Sound:   "default",
			Snoozed: false,
		},
		logger:      logger,
		soundQueue:  make(chan SoundRequest, 100), // Buffered channel
		rateLimiter: NewRateLimiter(),
		shutdownCh:  make(chan struct{}),
	}

	// Start the sound queue worker
	nm.wg.Add(1)
	go nm.soundWorker()

	return nm
}

// soundWorker processes sound requests sequentially
func (nm *NotificationManager) soundWorker() {
	defer nm.wg.Done()
	
	for {
		select {
		case <-nm.shutdownCh:
			return
		case req := <-nm.soundQueue:
			var err error
			if req.Type == "default" {
				err = nm.executeDefaultSound(req.ServiceName)
			} else {
				err = nm.executeCustomSound(req.SoundFile)
			}
			
			// Send result if channel provided
			if req.ResultChan != nil {
				select {
				case req.ResultChan <- err:
				case <-time.After(100 * time.Millisecond):
					// Don't block if receiver is not ready
				}
			}
		}
	}
}

// Shutdown gracefully stops the notification manager
func (nm *NotificationManager) Shutdown() {
	close(nm.shutdownCh)
	nm.wg.Wait()
}

func (nm *NotificationManager) GetConfig() NotificationConfig {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.config
}

func (nm *NotificationManager) SetEnabled(enabled bool) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.Enabled = enabled
	if nm.logger != nil {
		nm.logger.Info(fmt.Sprintf("Notifications enabled: %v", enabled))
	}
}

func (nm *NotificationManager) SetSound(sound string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// If it's not default and doesn't have an extension, try to find the file
	if sound != "default" && !strings.Contains(sound, ".") {
		soundsDir := filepath.Join(".", "assets", "sounds")
		entries, err := os.ReadDir(soundsDir)
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
				if nameWithoutExt == sound {
					sound = name // Use the full filename with extension
					break
				}
			}
		}
	}

	nm.config.Sound = sound
	nm.logger.Info(fmt.Sprintf("Notification sound set to: %s", sound))
}

func (nm *NotificationManager) SnoozeSound(minutes int) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.Snoozed = true
	nm.config.SnoozeUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
	if nm.logger != nil {
		nm.logger.Info(fmt.Sprintf("Sound snoozed for %d minutes", minutes))
	}
}

func (nm *NotificationManager) UnsnoozeSound() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.Snoozed = false
	nm.config.SnoozeUntil = time.Time{}
	if nm.logger != nil {
		nm.logger.Info("Sound unsnoozed")
	}
}

func (nm *NotificationManager) IsSnoozeActive() bool {
	nm.mu.RLock()
	snoozed := nm.config.Snoozed
	snoozeUntil := nm.config.SnoozeUntil
	nm.mu.RUnlock()

	if !snoozed {
		return false
	}

	// Check if snooze period has expired
	if time.Now().After(snoozeUntil) {
		nm.UnsnoozeSound()
		return false
	}

	return true
}


func (nm *NotificationManager) SendNotification(serviceSummary, message, htmlURL, serviceName string) error {
	nm.mu.RLock()
	config := nm.config
	nm.mu.RUnlock()

	if !config.Enabled {
		return nil
	}

	// Apply rate limiting
	if !nm.rateLimiter.Allow() {
		nm.logger.Warn("Notification rate limited - too many notifications")
		return nil
	}

	// Use terminal-notifier for macOS notifications with URL support
	args := []string{
		"-title", serviceSummary,  // Service summary as title
		"-message", message,        // Incident title as message  
	}

	// Add URL if provided - clicking notification will open the incident
	if htmlURL != "" {
		args = append(args, "-open", htmlURL)
	}

	cmd := exec.Command("terminal-notifier", args...)
	err := cmd.Run()
	if err != nil && nm.logger != nil {
		// Fallback to osascript if terminal-notifier is not installed
		fallbackCmd := exec.Command("osascript", "-e", 
			fmt.Sprintf(`display notification "%s" with title "%s"`, message, serviceSummary))
		if fallbackErr := fallbackCmd.Run(); fallbackErr != nil {
			nm.logger.Error(fmt.Sprintf("Failed to send notification: %v (fallback also failed: %v)", err, fallbackErr))
			return fmt.Errorf("notification failed: %w", err)
		}
	}

	// Queue sound playback if not snoozed
	if !nm.IsSnoozeActive() {
		soundReq := SoundRequest{
			Type:        "default",
			ServiceName: serviceName, // Use the configured service name
		}
		
		if config.Sound != "default" {
			soundReq.Type = "custom"
			soundReq.SoundFile = config.Sound
		}
		
		// Non-blocking send to queue
		select {
		case nm.soundQueue <- soundReq:
			// Queued successfully
		default:
			nm.logger.Warn("Sound queue full, skipping sound playback")
		}
	}

	return nil
}

// executeDefaultSound uses the say command with the configured service name
func (nm *NotificationManager) executeDefaultSound(serviceName string) error {
	if serviceName == "" {
		serviceName = "New Incident"
	}

	cmd := exec.Command("say", serviceName)
	err := cmd.Run()
	if err != nil && nm.logger != nil {
		nm.logger.Error(fmt.Sprintf("Failed to play default sound: %v", err))
		return err
	}
	return nil
}

// executeCustomSound uses afplay for custom sound files
func (nm *NotificationManager) executeCustomSound(soundFile string) error {
	soundPath := filepath.Join(".", "assets", "sounds", soundFile)

	// Check if file exists
	if _, err := os.Stat(soundPath); err != nil {
		nm.logger.Error(fmt.Sprintf("Sound file not found: %s", soundPath))
		return err
	}

	// Use afplay for macOS
	cmd := exec.Command("afplay", soundPath)
	err := cmd.Run()
	if err != nil && nm.logger != nil {
		nm.logger.Error(fmt.Sprintf("Failed to play custom sound %s: %v", soundPath, err))
		return err
	}
	return nil
}

func (nm *NotificationManager) GetAvailableSounds() ([]string, error) {
	soundsDir := filepath.Join(".", "assets", "sounds")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(soundsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sounds directory: %w", err)
	}

	sounds := []string{"default"} // Always include default option

	// Read sound files from directory
	entries, err := os.ReadDir(soundsDir)
	if err != nil {
		nm.logger.Warn(fmt.Sprintf("Failed to read sounds directory: %v", err))
		return sounds, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".mp3" || ext == ".wav" || ext == ".m4a" || ext == ".aiff" {
			// Remove extension for display
			nameWithoutExt := strings.TrimSuffix(name, ext)
			sounds = append(sounds, nameWithoutExt)
		}
	}

	return sounds, nil
}

func (nm *NotificationManager) TestSound() error {
	nm.mu.RLock()
	sound := nm.config.Sound
	nm.mu.RUnlock()

	// Create a request with result channel for testing
	resultChan := make(chan error, 1)
	
	soundReq := SoundRequest{
		Type:        "default",
		ServiceName: "Test Notification",
		ResultChan:  resultChan,
	}
	
	if sound != "default" {
		soundReq.Type = "custom"
		soundReq.SoundFile = sound
	}

	// Send to queue
	select {
	case nm.soundQueue <- soundReq:
		// Wait for result with timeout
		select {
		case err := <-resultChan:
			return err
		case <-time.After(5 * time.Second):
			return fmt.Errorf("sound playback timeout")
		}
	default:
		return fmt.Errorf("sound queue is full")
	}
}