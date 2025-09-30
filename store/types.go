package store

import "time"

// ServiceConfig represents a single service configuration
type ServiceConfig struct {
	ID       interface{} `json:"id"`
	Name     string      `json:"name"`
	Disabled bool        `json:"disabled,omitempty"` // Added to track disabled state
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
// IncidentAlert represents alert data for an incident
type IncidentAlert struct {
	ID          string      `json:"id"`
	Summary     string      `json:"summary"`
	Status      string      `json:"status"`
	CreatedAt   string      `json:"created_at"`
	ServiceName string      `json:"service_name,omitempty"`
	Links       []AlertLink `json:"links,omitempty"`
}

// AlertLink represents a link in an alert
type AlertLink struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// IncidentNote represents a note on an incident
type IncidentNote struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UserName  string `json:"user_name,omitempty"`
}

// IncidentSidebarData represents the complete sidebar data for an incident
type IncidentSidebarData struct {
	IncidentID string          `json:"incident_id"`
	Alerts     []IncidentAlert `json:"alerts"`
	Notes      []IncidentNote  `json:"notes"`
	Loading    bool            `json:"loading"`
	Error      string          `json:"error,omitempty"`
}