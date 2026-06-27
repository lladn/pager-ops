package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// pdAPIBase is the PagerDuty REST API base URL. go-pagerduty v1.8.0 does not
// expose incident custom fields, so those endpoints are called directly.
const pdAPIBase = "https://api.pagerduty.com"

// incidentType is the incident type whose custom fields are managed. PagerDuty
// uses "incident_default" for the standard incident type.
const incidentType = "incident_default"

// CustomFieldOption represents a single selectable value for a fixed-option custom field.
type CustomFieldOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// CustomField represents an incident custom field definition merged with the
// value currently set on a specific incident.
type CustomField struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	DataType    string              `json:"data_type"`  // e.g. "string"
	FieldType   string              `json:"field_type"` // e.g. "single_value_fixed", "multi_value_fixed"
	Options     []CustomFieldOption `json:"options"`
	Value       interface{}         `json:"value"` // current value on the incident (string, []string, or nil)
}

// IsMultiValue reports whether the field accepts more than one value.
func (f CustomField) IsMultiValue() bool {
	return f.FieldType == "multi_value" || f.FieldType == "multi_value_fixed"
}

// SetCustomFieldValueRequest represents a request to set a custom field value on an incident.
type SetCustomFieldValueRequest struct {
	IncidentID string
	FieldID    string
	Value      interface{} // string for single-value fields, []string for multi-value fields
	From       string      // acting user's email, sent as the PagerDuty "From" header
}

// CustomFieldValue represents the value currently set for a custom field on an
// incident, as returned by GET /incidents/{id}/custom_fields/values.
type CustomFieldValue struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Value       interface{} `json:"value"`
}

// --- raw HTTP helper -------------------------------------------------------

// pdRequest performs an authenticated request against the PagerDuty REST API
// and returns the raw response body. Non-2xx responses are returned as errors.
func (c *Client) pdRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, pdAPIBase+path, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Token token="+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pagerduty API %s %s returned %d: %s", method, path, resp.StatusCode, string(data))
	}

	return data, nil
}

// --- field definitions + values -------------------------------------------

// fetchIncidentCustomFields retrieves the custom field definitions (with their
// fixed options) and merges in the values currently set on the given incident.
func (c *Client) fetchIncidentCustomFields(ctx context.Context, incidentID string) ([]CustomField, error) {
	// 1. Field definitions with their selectable options.
	defsData, err := c.pdRequest(ctx, "GET",
		fmt.Sprintf("/incidents/types/%s/custom_fields?include[]=field_options", incidentType), nil, nil)
	if err != nil {
		return nil, err
	}

	var defsResp struct {
		Fields []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			DisplayName  string `json:"display_name"`
			DataType     string `json:"data_type"`
			FieldType    string `json:"field_type"`
			FieldOptions []struct {
				ID   string `json:"id"`
				Data struct {
					Value string `json:"value"`
				} `json:"data"`
			} `json:"field_options"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(defsData, &defsResp); err != nil {
		return nil, fmt.Errorf("failed to parse custom field definitions: %w", err)
	}

	// 2. Current values set on this incident, keyed by field id.
	values := make(map[string]interface{})
	if incidentID != "" {
		valsData, err := c.pdRequest(ctx, "GET",
			fmt.Sprintf("/incidents/%s/custom_fields/values", incidentID), nil, nil)
		if err != nil {
			// Don't fail the whole request if values can't be read; surface defs only.
			c.logger(fmt.Sprintf("failed to fetch custom field values for %s: %v", incidentID, err))
		} else {
			var valsResp struct {
				CustomFields []struct {
					ID    string      `json:"id"`
					Name  string      `json:"name"`
					Value interface{} `json:"value"`
				} `json:"custom_fields"`
			}
			if err := json.Unmarshal(valsData, &valsResp); err != nil {
				c.logger(fmt.Sprintf("failed to parse custom field values for %s: %v", incidentID, err))
			} else {
				for _, v := range valsResp.CustomFields {
					values[v.ID] = v.Value
				}
			}
		}
	}

	fields := make([]CustomField, 0, len(defsResp.Fields))
	for _, f := range defsResp.Fields {
		field := CustomField{
			ID:          f.ID,
			Name:        f.Name,
			DisplayName: f.DisplayName,
			DataType:    f.DataType,
			FieldType:   f.FieldType,
			Value:       values[f.ID],
		}
		for _, opt := range f.FieldOptions {
			field.Options = append(field.Options, CustomFieldOption{
				ID:    opt.ID,
				Value: opt.Data.Value,
			})
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// fetchIncidentCustomFieldValues retrieves only the values currently set on an
// incident (GET /incidents/{id}/custom_fields/values).
func (c *Client) fetchIncidentCustomFieldValues(ctx context.Context, incidentID string) ([]CustomFieldValue, error) {
	data, err := c.pdRequest(ctx, "GET",
		fmt.Sprintf("/incidents/%s/custom_fields/values", incidentID), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		CustomFields []CustomFieldValue `json:"custom_fields"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse custom field values: %w", err)
	}

	return resp.CustomFields, nil
}

// putIncidentCustomFieldValue sets a single custom field value on an incident.
func (c *Client) putIncidentCustomFieldValue(ctx context.Context, req SetCustomFieldValueRequest) error {
	body := map[string]interface{}{
		"custom_fields": []map[string]interface{}{
			{
				"id":    req.FieldID,
				"value": req.Value,
			},
		},
	}

	// PagerDuty requires a "From" header (acting user's email) for incident write
	// operations, including setting custom field values.
	headers := map[string]string{"From": req.From}

	_, err := c.pdRequest(ctx, "PUT",
		fmt.Sprintf("/incidents/%s/custom_fields/values", req.IncidentID), body, headers)
	return err
}

// --- public, queue-fronted methods ----------------------------------------

// GetIncidentCustomFields returns the incident custom field definitions merged
// with the values currently set on the given incident, via the rate-limited queue.
func (c *Client) GetIncidentCustomFields(incidentID string) ([]CustomField, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.queueRequest("GetCustomFields", ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incident custom fields: %w", err)
	}

	fields, ok := result.([]CustomField)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for custom fields")
	}

	return fields, nil
}

// GetIncidentCustomFieldValues returns the values currently set on the incident,
// via the rate-limited queue.
func (c *Client) GetIncidentCustomFieldValues(incidentID string) ([]CustomFieldValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.queueRequest("GetCustomFieldValues", ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incident custom field values: %w", err)
	}

	values, ok := result.([]CustomFieldValue)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for custom field values")
	}

	return values, nil
}

// SetIncidentCustomFieldValue sets a custom field value on an incident via the queue.
// fromEmail is the acting user's email, required by PagerDuty as the "From" header.
func (c *Client) SetIncidentCustomFieldValue(incidentID, fieldID string, value interface{}, fromEmail string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := SetCustomFieldValueRequest{
		IncidentID: incidentID,
		FieldID:    fieldID,
		Value:      value,
		From:       fromEmail,
	}

	_, err := c.queueRequest("SetCustomFieldValue", ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to set custom field value: %w", err)
	}

	return nil
}
