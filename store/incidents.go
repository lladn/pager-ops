package store

import (
	"context"
	"time"
	"github.com/PagerDuty/go-pagerduty"
	"pager-ops/database"
)

func (c *Client) FetchOpenIncidents(
	serviceIDs []string, 
	userID string) (
	[]database.IncidentData, error,
) {
	var allIncidents []database.IncidentData

	// Fetch incidents filtered by services
	if len(serviceIDs) > 0 {
		serviceIncidents, err := c.fetchIncidentsByServices(
			serviceIDs, []string{"triggered", "acknowledged"})
		if err != nil {
			return nil, err
		}
		allIncidents = append(allIncidents, serviceIncidents...)
	}

	// Fetch incidents assigned to current user
	if userID != "" {
		userIncidents, err := c.fetchIncidentsByUser(
			userID, []string{"triggered", "acknowledged"})
		if err != nil {
			return nil, err
		}
		allIncidents = append(allIncidents, userIncidents...)
	}

	// Deduplicate incidents
	return deduplicateIncidents(allIncidents), nil
}

func (c *Client) FetchResolvedIncidents(
	serviceIDs []string) (
	[]database.IncidentData, error,
) {
	// Calculate date range for last week
	until := time.Now()
	since := until.AddDate(0, 0, -7)

	opts := pagerduty.ListIncidentsOptions{
		Statuses:   []string{"resolved"},
		ServiceIDs: serviceIDs,
		Since:      since.Format(time.RFC3339),
		Until:      until.Format(time.RFC3339),
		Limit:      100,
		SortBy:     "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)

	for {
		opts.Offset = offset
		resp, err := c.pd.ListIncidentsWithContext(context.TODO(), opts)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Incidents {
			incidents := convertToIncidentData(i)
			allIncidents = append(allIncidents, incidents)
		}

		if !resp.More {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

func (c *Client) fetchIncidentsByServices(
	serviceIDs []string, 
	statuses []string) (
	[]database.IncidentData, error,
) {
	opts := pagerduty.ListIncidentsOptions{
		Statuses:   statuses,
		ServiceIDs: serviceIDs,
		Limit:      100,
		SortBy:     "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)

	for {
		opts.Offset = offset
		resp, err := c.pd.ListIncidentsWithContext(context.TODO(), opts)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Incidents {
			incidents := convertToIncidentData(i)
			allIncidents = append(allIncidents, incidents)
		}

		if !resp.More {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

func (c *Client) fetchIncidentsByUser(
	userID string, 
	statuses []string) (
	[]database.IncidentData, error,
) {
	opts := pagerduty.ListIncidentsOptions{
		Statuses: statuses,
		UserIDs:  []string{userID},
		Limit:    100,
		SortBy:   "created_at:desc",
	}

	var allIncidents []database.IncidentData
	offset := uint(0)

	for {
		opts.Offset = offset
		resp, err := c.pd.ListIncidentsWithContext(context.TODO(), opts)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Incidents {
			incidents := convertToIncidentData(i)
			allIncidents = append(allIncidents, incidents)
		}

		if !resp.More {
			break
		}
		offset += opts.Limit
	}

	return allIncidents, nil
}

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