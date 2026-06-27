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

// ResolveIncident resolves an incident through the queue
func (c *Client) ResolveIncident(incidentID, userEmail string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := ManageIncidentsRequest{
		From:       userEmail,
		IncidentID: incidentID,
		Status:     "resolved",
	}

	result, err := c.queueRequest("ManageIncidents", ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to resolve incident: %w", err)
	}

	// Check if the response indicates success
	if result != nil {
		return nil
	}

	return fmt.Errorf("unexpected response from resolve incident")
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

// FormatNoteContent converts structured note data into a single formatted string.
// The layout is always deterministic, whether or not tags are present:
//   1. first answered question
//   2. tag groups with selections (in config order)
//   3. remaining answered questions (in config order)
//   4. freeform content
// Empty fields are excluded.
func FormatNoteContent(responses []NoteResponse, tags []NoteTag, freeformContent string) string {
	var parts []string

	appendResponse := func(r NoteResponse) {
		if strings.TrimSpace(r.Answer) != "" {
			parts = append(parts, r.Question)
			parts = append(parts, r.Answer)
			parts = append(parts, "") // Blank line after each Q&A
		}
	}

	// 1. First question
	if len(responses) > 0 {
		appendResponse(responses[0])
	}

	// 2. Tag groups, in order
	for _, tag := range tags {
		if len(tag.SelectedValues) > 0 {
			parts = append(parts, fmt.Sprintf("%s:", tag.TagName))
			parts = append(parts, tag.SelectedValues...)
			parts = append(parts, "") // Blank line after each tag group
		}
	}

	// 3. Remaining questions, in order
	for i := 1; i < len(responses); i++ {
		appendResponse(responses[i])
	}

	// 4. Freeform content at the end
	if strings.TrimSpace(freeformContent) != "" {
		parts = append(parts, strings.TrimSpace(freeformContent))
	}

	// Join all parts with newlines
	result := strings.Join(parts, "\n")
	return strings.TrimSpace(result)
}