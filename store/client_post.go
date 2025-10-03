package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AcknowledgeIncident acknowledges an incident through the queue
func (c *Client) AcknowledgeIncident(incidentID, userEmail string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := ManageIncidentsRequest{
		From:       userEmail,
		IncidentID: incidentID,
		Status:     "acknowledged",
	}

	result, err := c.queueRequest("ManageIncidents", ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to acknowledge incident: %w", err)
	}

	// Check if the response indicates success
	if result != nil {
		return nil
	}

	return fmt.Errorf("unexpected response from acknowledge incident")
}

// CreateIncidentNote creates a note on an incident through the queue
func (c *Client) CreateIncidentNote(incidentID string, noteContent string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := CreateIncidentNoteRequest{
		IncidentID: incidentID,
		Content:    noteContent,
	}

	result, err := c.queueRequest("CreateIncidentNote", ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create incident note: %w", err)
	}

	// Check if the response indicates success
	if result != nil {
		return nil
	}

	return fmt.Errorf("unexpected response from create incident note")
}

// ManageIncidentsRequest represents options for managing incidents
type ManageIncidentsRequest struct {
	From       string
	IncidentID string
	Status     string
}

// CreateIncidentNoteRequest represents options for creating a note
type CreateIncidentNoteRequest struct {
	IncidentID string
	Content    string
}

// FormatNoteContent converts structured note data into a single formatted string
// Empty fields are excluded from the output
func FormatNoteContent(responses []NoteResponse, tags []NoteTag, freeformContent string) string {
	var parts []string

	// Add question responses
	for _, response := range responses {
		if strings.TrimSpace(response.Answer) != "" {
			parts = append(parts, fmt.Sprintf("%s\n* %s\n", response.Question, response.Answer))
		}
	}

	// Add tags
	for _, tag := range tags {
		if len(tag.SelectedValues) > 0 {
			parts = append(parts, fmt.Sprintf("%s:", tag.TagName))
			for _, value := range tag.SelectedValues {
				parts = append(parts, value)
			}
			parts = append(parts, "") // Empty line after tag group
		}
	}

	// Add freeform content at the end
	if strings.TrimSpace(freeformContent) != "" {
		parts = append(parts, strings.TrimSpace(freeformContent))
	}

	// Join all parts with newlines
	result := strings.Join(parts, "\n")
	return strings.TrimSpace(result)
}
