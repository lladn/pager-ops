package store

import "time"

// ServiceConfig represents a single service configuration
type ServiceConfig struct {
	ID   interface{} `json:"id"`
	Name string      `json:"name"`
}

// ServicesConfig represents the overall services configuration
type ServicesConfig struct {
	Services []ServiceConfig `json:"services"`
}

// IncidentSummary represents a summary of an incident
type IncidentSummary struct {
	IncidentID     string    `json:"incident_id"`
	IncidentNumber int       `json:"incident_number"`
	Title          string    `json:"title"`
	ServiceSummary string    `json:"service_summary"`
	ServiceID      string    `json:"service_id"`
	Status         string    `json:"status"`
	HTMLURL        string    `json:"html_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AlertCount     int       `json:"alert_count"`
}