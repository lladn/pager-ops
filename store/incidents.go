package store

import (
	"pager-ops/database"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

func convertToIncidentData(
	i pagerduty.Incident) database.IncidentData {
	// IncidentNumber is already a uint in PagerDuty API
	incidentNum := int(i.IncidentNumber)

	createdAtTime, _ := time.Parse(time.RFC3339, i.CreatedAt)
	updatedAtTime, _ := time.Parse(time.RFC3339, i.LastStatusChangeAt)

	// AlertCounts.All is uint, convert to int
	alertCount := 0
	if i.AlertCounts.All > 0 {
		alertCount = int(i.AlertCounts.All)
	}

	// Service is an APIObject, check if ID is not empty
	serviceSummary := ""
	serviceID := ""
	if i.Service.ID != "" {
		serviceSummary = i.Service.Summary
		serviceID = i.Service.ID
	}

	// Extract urgency from incident
	urgency := "low"
	if i.Urgency != "" {
		urgency = i.Urgency
	}

	// Collect the names of everyone who acknowledged the incident.
	ackNames := make([]string, 0, len(i.Acknowledgements))
	for _, ack := range i.Acknowledgements {
		name := ack.Acknowledger.Summary
		if name != "" {
			ackNames = append(ackNames, name)
		}
	}
	acknowledgedBy := strings.Join(ackNames, ", ")

	return database.IncidentData{
		IncidentID:     i.ID,
		IncidentNumber: incidentNum,
		Title:          i.Title,
		ServiceSummary: serviceSummary,
		ServiceID:      serviceID,
		Status:         i.Status,
		HTMLURL:        i.HTMLURL,
		CreatedAt:      createdAtTime,
		UpdatedAt:      updatedAtTime,
		AlertCount:     alertCount,
		Urgency:        urgency,
		AcknowledgedBy: acknowledgedBy,
	}
}

func deduplicateIncidents(
	incidents []database.IncidentData) []database.IncidentData {
	seen := make(map[string]bool)
	result := []database.IncidentData{}

	for _, incident := range incidents {
		if !seen[incident.IncidentID] {
			seen[incident.IncidentID] = true
			result = append(result, incident)
		}
	}

	return result
}
