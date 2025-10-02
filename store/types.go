package store

import "time"

// TagConfig represents a tag configuration for a service
type TagConfig struct {
	Name     string   `json:"name"`
	Multiple []string `json:"multiple,omitempty"` // Multiple selection tags
	Single   []string `json:"single,omitempty"`   // Single selection tags
}

// ServiceTypes represents the types configuration for a service
type ServiceTypes struct {
	Questions []string    `json:"questions,omitempty"` // Optional questions
	Tags      []TagConfig `json:"tags,omitempty"`      // Optional tags
}

// ServiceConfig represents a single service configuration
type ServiceConfig struct {
	ID       interface{}   `json:"id"`
	Name     string        `json:"name"`
	Disabled bool          `json:"disabled,omitempty"` // Added to track disabled state
	Types    *ServiceTypes `json:"types,omitempty"`    // Optional notekit configuration
}

// ServicesConfig represents the overall services configuration
type ServicesConfig struct {
	Services []ServiceConfig `json:"services"`
}

// NoteResponse represents a single question-answer pair
type NoteResponse struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// NoteTag represents tag selections for a note
type NoteTag struct {
	TagName        string   `json:"tag_name"`
	SelectedValues []string `json:"selected_values"` // 1 item for single, N for multiple
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
	ID              string         `json:"id"`
	Content         string         `json:"content"`                    // Freeform content
	CreatedAt       string         `json:"created_at"`
	UserName        string         `json:"user_name,omitempty"`
	ServiceID       string         `json:"service_id,omitempty"`       // Service reference
	Responses       []NoteResponse `json:"responses,omitempty"`        // Structured Q&A
	Tags            []NoteTag      `json:"tags,omitempty"`             // Tag selections
	FreeformContent string         `json:"freeform_content,omitempty"` // Additional freeform text
}

// IncidentSidebarData represents the complete sidebar data for an incident
type IncidentSidebarData struct {
	IncidentID string          `json:"incident_id"`
	Alerts     []IncidentAlert `json:"alerts"`
	Notes      []IncidentNote  `json:"notes"`
	Loading    bool            `json:"loading"`
	Error      string          `json:"error,omitempty"`
}