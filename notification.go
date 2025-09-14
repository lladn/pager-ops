// notification.go - CORRECTED VERSION
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

type NotificationConfig struct {
	Enabled     bool      `json:"enabled"`
	Sound       string    `json:"sound"`
	Snoozed     bool      `json:"snoozed"`
	SnoozeUntil time.Time `json:"snoozeUntil"`
}

type NotificationManager struct {
	config      NotificationConfig
	mu          sync.RWMutex
	logger      *Logger
	soundPlayer *SoundPlayer
}

type SoundPlayer struct {
	initialized bool
	initOnce    sync.Once 
	initErr     error    
}

var speakerInitMutex sync.Mutex

func NewNotificationManager(logger *Logger) *NotificationManager {
	return &NotificationManager{
		config: NotificationConfig{
			Enabled: true,
			Sound:   "default",
			Snoozed: false,
		},
		logger:      logger,
		soundPlayer: &SoundPlayer{},
	}
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

// Update SetSound to handle names without extensions
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
	defer nm.mu.RUnlock()

	if !nm.config.Snoozed {
		return false
	}

	// Check if snooze period has expired
	if time.Now().After(nm.config.SnoozeUntil) {
		// Create a goroutine to handle the unsnooze to avoid deadlock
		go nm.UnsnoozeSound()
		return false
	}

	return true
}

func (nm *NotificationManager) SendNotification(title, message, serviceName string) error {
	nm.mu.RLock()
	config := nm.config
	nm.mu.RUnlock()

	if !config.Enabled {
		return nil
	}

	// Send desktop notification
	err := beeep.Notify(title, message, "")
	if err != nil && nm.logger != nil {
		nm.logger.Error(fmt.Sprintf("Failed to send notification: %v", err))
	}

	// Play sound if not snoozed
	if !nm.IsSnoozeActive() {
		if config.Sound == "default" {
			nm.playDefaultSound(serviceName)
		} else {
			nm.playCustomSound(config.Sound)
		}
	}

	return err
}

func (nm *NotificationManager) playDefaultSound(serviceName string) {
	if serviceName == "" {
		serviceName = "New Incident"
	}

	cmd := exec.Command("say", serviceName)
	err := cmd.Run()
	if err != nil && nm.logger != nil {
		nm.logger.Error(fmt.Sprintf("Failed to play default sound: %v", err))
	}
}

func (nm *NotificationManager) playCustomSound(soundFile string) {
	soundPath := filepath.Join(".", "assets", "sounds", soundFile)

	// Open the sound file first
	f, err := os.Open(soundPath)
	if err != nil {
		nm.logger.Error(fmt.Sprintf("Failed to open sound file %s: %v", soundPath, err))
		return
	}
	defer f.Close()

	var stream beep.StreamSeekCloser
	var format beep.Format

	// Decode based on file extension
	ext := strings.ToLower(filepath.Ext(soundFile))
	switch ext {
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		nm.logger.Error(fmt.Sprintf("Unsupported audio format: %s", ext))
		return
	}

	if err != nil {
		nm.logger.Error(fmt.Sprintf("Failed to decode sound file: %v", err))
		return
	}
	defer stream.Close()

	// Use global mutex for speaker initialization to prevent race
	speakerInitMutex.Lock()
	if !nm.soundPlayer.initialized {
		nm.soundPlayer.initOnce.Do(func() {
			nm.soundPlayer.initErr = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			if nm.soundPlayer.initErr == nil {
				nm.soundPlayer.initialized = true
				nm.logger.Info("Speaker initialized successfully")
			}
		})

		if nm.soundPlayer.initErr != nil {
			speakerInitMutex.Unlock()
			nm.logger.Error(fmt.Sprintf("Failed to initialize speaker: %v", nm.soundPlayer.initErr))
			return
		}
	}
	speakerInitMutex.Unlock()

	// Play the sound
	done := make(chan bool)
	speaker.Play(beep.Seq(stream, beep.Callback(func() {
		done <- true
	})))

	// Wait for playback to complete (with timeout)
	select {
	case <-done:
		// Sound finished playing
	case <-time.After(10 * time.Second):
		// Timeout after 10 seconds
		nm.logger.Warn("Sound playback timeout")
	}
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
		if ext == ".mp3" || ext == ".wav" {
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

	if sound == "default" {
		nm.playDefaultSound("Test Notification")
	} else {
		nm.playCustomSound(sound)
	}

	return nil
}
