package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection - NO CHANGES TO EXISTING STRUCT
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex // Added for thread safety
}

// IncidentData represents an incident from PagerDuty - NO CHANGES TO EXISTING STRUCT
type IncidentData struct {
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
	Urgency        string    `json:"urgency"`
}

// SidebarAlert represents alert data stored in database
type SidebarAlert struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	ServiceName string `json:"service_name,omitempty"`
	Links       string `json:"links,omitempty"` // JSON string
}

// SidebarNote represents note data stored in database  // SidebarNote represents note data stored in database  
type SidebarNote struct {
	ID              string `json:"id"`
	Content         string `json:"content"`
	CreatedAt       string `json:"created_at"`
	UserName        string `json:"user_name,omitempty"`
	ServiceID       string `json:"service_id,omitempty"`
	Responses       string `json:"responses,omitempty"`        // JSON string
	Tags            string `json:"tags,omitempty"`             // JSON string
	FreeformContent string `json:"freeform_content,omitempty"`
}

// SidebarMetadata represents metadata for sidebar data
type SidebarMetadata struct {
	IncidentID        string
	LastFetchedAlerts *time.Time
	LastFetchedNotes  *time.Time
	LastAlertCount    int
	LastUpdatedAt     *time.Time
}

// NewDB creates a new database connection - ORIGINAL METHOD UNCHANGED
func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}

	// Create tables if they don't exist
	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, err
	}
	
	// Create sidebar tables
	if err := db.createSidebarTables(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}


// StoreIncidentAlerts stores alerts for an incident (links already JSON)
func (db *DB) StoreIncidentAlerts(incidentID string, alerts []SidebarAlert) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete existing alerts for the incident
	_, err = tx.Exec("DELETE FROM incident_alerts WHERE incident_id = ?", incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete existing alerts: %w", err)
	}
	
	// Prepare insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO incident_alerts (id, incident_id, summary, status, created_at, service_name, links)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert new alerts
	for _, alert := range alerts {
		_, err = stmt.Exec(
			alert.ID,
			incidentID,
			alert.Summary,
			alert.Status,
			alert.CreatedAt,
			alert.ServiceName,
			alert.Links, // Already JSON string
		)
		if err != nil {
			return fmt.Errorf("failed to insert alert %s: %w", alert.ID, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

func (db *DB) GetIncidentAlerts(incidentID string) ([]SidebarAlert, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	query := `
		SELECT id, summary, status, created_at, service_name, links
		FROM incident_alerts
		WHERE incident_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := db.conn.Query(query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()
	
	var alerts []SidebarAlert
	for rows.Next() {
		var alert SidebarAlert
		
		err := rows.Scan(
			&alert.ID,
			&alert.Summary,
			&alert.Status,
			&alert.CreatedAt,
			&alert.ServiceName,
			&alert.Links, // JSON string
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		
		alerts = append(alerts, alert)
	}
	
	return alerts, nil
}

func (db *DB) StoreIncidentNotes(incidentID string, notes []SidebarNote) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete existing notes for the incident
	_, err = tx.Exec("DELETE FROM incident_notes WHERE incident_id = ?", incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete existing notes: %w", err)
	}
	
	// Prepare insert statement with enhanced fields
	stmt, err := tx.Prepare(`
		INSERT INTO incident_notes (id, incident_id, content, created_at, user_name, service_id, responses, tags, freeform_content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert new notes
	for _, note := range notes {
		_, err = stmt.Exec(
			note.ID,
			incidentID,
			note.Content,
			note.CreatedAt,
			note.UserName,
			note.ServiceID,
			note.Responses,
			note.Tags,
			note.FreeformContent,
		)
		if err != nil {
			return fmt.Errorf("failed to insert note %s: %w", note.ID, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

func (db *DB) GetIncidentNotes(incidentID string) ([]SidebarNote, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	query := `
		SELECT id, content, created_at, user_name, service_id, responses, tags, freeform_content
		FROM incident_notes
		WHERE incident_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := db.conn.Query(query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()
	
	var notes []SidebarNote
	for rows.Next() {
		var note SidebarNote
		var serviceID, responses, tags, freeformContent sql.NullString
		
		err := rows.Scan(
			&note.ID,
			&note.Content,
			&note.CreatedAt,
			&note.UserName,
			&serviceID,
			&responses,
			&tags,
			&freeformContent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		
		// Handle nullable fields
		if serviceID.Valid {
			note.ServiceID = serviceID.String
		}
		if responses.Valid {
			note.Responses = responses.String
		}
		if tags.Valid {
			note.Tags = tags.String
		}
		if freeformContent.Valid {
			note.FreeformContent = freeformContent.String
		}
		
		notes = append(notes, note)
	}
	
	return notes, nil
}


// GetSidebarMetadata retrieves metadata for sidebar data
func (db *DB) GetSidebarMetadata(incidentID string) (*SidebarMetadata, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	query := `
		SELECT last_fetched_alerts, last_fetched_notes, last_alert_count, last_updated_at
		FROM incident_sidebar_metadata
		WHERE incident_id = ?
	`
	
	var metadata SidebarMetadata
	var lastFetchedAlerts, lastFetchedNotes, lastUpdatedAt sql.NullTime
	
	err := db.conn.QueryRow(query, incidentID).Scan(
		&lastFetchedAlerts,
		&lastFetchedNotes,
		&metadata.LastAlertCount,
		&lastUpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No metadata exists
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}
	
	metadata.IncidentID = incidentID
	if lastFetchedAlerts.Valid {
		metadata.LastFetchedAlerts = &lastFetchedAlerts.Time
	}
	if lastFetchedNotes.Valid {
		metadata.LastFetchedNotes = &lastFetchedNotes.Time
	}
	if lastUpdatedAt.Valid {
		metadata.LastUpdatedAt = &lastUpdatedAt.Time
	}
	
	return &metadata, nil
}

func (db *DB) UpdateSidebarMetadata(incidentID string, alertCount int, updatedAt time.Time, fetchedAlerts bool, fetchedNotes bool) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	// Get current metadata to preserve unfetched timestamps
	var existingAlertsFetch, existingNotesFetch sql.NullTime
	
	query := `SELECT last_fetched_alerts, last_fetched_notes FROM incident_sidebar_metadata WHERE incident_id = ?`
	err := db.conn.QueryRow(query, incidentID).Scan(&existingAlertsFetch, &existingNotesFetch)
	
	now := time.Now()
	var alertsFetch, notesFetch sql.NullTime
	
	if err == sql.ErrNoRows {
		// No existing metadata, set times based on what was fetched
		if fetchedAlerts {
			alertsFetch = sql.NullTime{Time: now, Valid: true}
		}
		if fetchedNotes {
			notesFetch = sql.NullTime{Time: now, Valid: true}
		}
	} else if err == nil {
		// Preserve existing timestamps, update only what was fetched
		alertsFetch = existingAlertsFetch
		notesFetch = existingNotesFetch
		
		if fetchedAlerts {
			alertsFetch = sql.NullTime{Time: now, Valid: true}
		}
		if fetchedNotes {
			notesFetch = sql.NullTime{Time: now, Valid: true}
		}
	} else {
		return fmt.Errorf("failed to query existing metadata: %w", err)
	}
	
	// Upsert the metadata
	upsertQuery := `
		INSERT INTO incident_sidebar_metadata (incident_id, last_fetched_alerts, last_fetched_notes, last_alert_count, last_updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(incident_id) DO UPDATE SET
			last_fetched_alerts = excluded.last_fetched_alerts,
			last_fetched_notes = excluded.last_fetched_notes,
			last_alert_count = excluded.last_alert_count,
			last_updated_at = excluded.last_updated_at
	`
	
	_, err = db.conn.Exec(upsertQuery, incidentID, alertsFetch, notesFetch, alertCount, updatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	
	return nil
}

func (db *DB) CleanupOldSidebarData(cutoffDate time.Time) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete alerts for old incidents
	_, err = tx.Exec(`
		DELETE FROM incident_alerts
		WHERE incident_id IN (
			SELECT incident_id FROM incidents
			WHERE updated_at < ?
		)
	`, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old alerts: %w", err)
	}
	
	// Delete notes for old incidents
	_, err = tx.Exec(`
		DELETE FROM incident_notes
		WHERE incident_id IN (
			SELECT incident_id FROM incidents
			WHERE updated_at < ?
		)
	`, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old notes: %w", err)
	}
	
	// Delete metadata for old incidents
	_, err = tx.Exec(`
		DELETE FROM incident_sidebar_metadata
		WHERE incident_id IN (
			SELECT incident_id FROM incidents
			WHERE updated_at < ?
		)
	`, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old metadata: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit cleanup transaction: %w", err)
	}
	
	return nil
}


// createTables - ORIGINAL METHOD ENHANCED WITH INDEXES
func (db *DB) createTables() error {
	// Create incidents table with indexes for performance
	incidentsTable := `
	CREATE TABLE IF NOT EXISTS incidents (
		incident_id TEXT PRIMARY KEY,
		incident_number INTEGER,
		title TEXT,
		service_summary TEXT,
		service_id TEXT,
		status TEXT,
		html_url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		alert_count INTEGER DEFAULT 0,
		urgency TEXT DEFAULT 'low',
		UNIQUE(incident_id)
	);

	CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
	CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_id);
	CREATE INDEX IF NOT EXISTS idx_incidents_created ON incidents(created_at);
	CREATE INDEX IF NOT EXISTS idx_incidents_updated ON incidents(updated_at);
	`

	if _, err := db.conn.Exec(incidentsTable); err != nil {
		return fmt.Errorf("failed to create incidents table: %w", err)
	}

	return nil
}

// createSidebarTables creates tables for incident sidebar data
func (db *DB) createSidebarTables() error {
	// Create incident_alerts table
	alertsTable := `
	CREATE TABLE IF NOT EXISTS incident_alerts (
		id TEXT PRIMARY KEY,
		incident_id TEXT NOT NULL,
		summary TEXT,
		status TEXT,
		created_at TEXT,
		service_name TEXT,
		links TEXT,  -- JSON serialized array
		FOREIGN KEY (incident_id) REFERENCES incidents(incident_id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_alerts_incident ON incident_alerts(incident_id);
	`
	
	// Create incident_notes table with enhanced schema for notekit
	notesTable := `
	CREATE TABLE IF NOT EXISTS incident_notes (
		id TEXT PRIMARY KEY,
		incident_id TEXT NOT NULL,
		content TEXT,
		created_at TEXT,
		user_name TEXT,
		service_id TEXT,
		responses TEXT,
		tags TEXT,
		freeform_content TEXT,
		FOREIGN KEY (incident_id) REFERENCES incidents(incident_id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_notes_incident ON incident_notes(incident_id);
	CREATE INDEX IF NOT EXISTS idx_notes_service ON incident_notes(service_id);
	`
	
	// Create incident_sidebar_metadata table
	metadataTable := `
	CREATE TABLE IF NOT EXISTS incident_sidebar_metadata (
		incident_id TEXT PRIMARY KEY,
		last_fetched_alerts DATETIME,
		last_fetched_notes DATETIME,
		last_alert_count INTEGER DEFAULT 0,
		last_updated_at DATETIME,
		FOREIGN KEY (incident_id) REFERENCES incidents(incident_id) ON DELETE CASCADE
	);
	`
	
	// Execute all table creations
	if _, err := db.conn.Exec(alertsTable); err != nil {
		return fmt.Errorf("failed to create incident_alerts table: %w", err)
	}
	
	if _, err := db.conn.Exec(notesTable); err != nil {
		return fmt.Errorf("failed to create incident_notes table: %w", err)
	}
	
	if _, err := db.conn.Exec(metadataTable); err != nil {
		return fmt.Errorf("failed to create incident_sidebar_metadata table: %w", err)
	}
	
	return nil
}

// NEW METHOD - InitStateTable creates the state persistence table
func (db *DB) InitStateTable() error {
	stateTable := `
	CREATE TABLE IF NOT EXISTS app_state (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.conn.Exec(stateTable)
	if err != nil {
		return fmt.Errorf("failed to create state table: %w", err)
	}

	return nil
}

// NEW METHOD - SetState stores a key-value pair in the state table
func (db *DB) SetState(key, value string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		INSERT INTO app_state (key, value, updated_at) 
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET 
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.conn.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set state %s: %w", key, err)
	}

	return nil
}

// NEW METHOD - GetState retrieves a value from the state table
func (db *DB) GetState(key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var value string
	query := `SELECT value FROM app_state WHERE key = ?`

	err := db.conn.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("state key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get state %s: %w", key, err)
	}

	return value, nil
}

// UpsertIncident - ENHANCED WITH THREAD SAFETY, SIGNATURE UNCHANGED
func (db *DB) UpsertIncident(incident IncidentData) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Use REPLACE for SQLite upsert pattern
	query := `
		REPLACE INTO incidents (
			incident_id, incident_number, title, service_summary, 
			service_id, status, html_url, created_at, updated_at, 
			alert_count, urgency
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query,
		incident.IncidentID,
		incident.IncidentNumber,
		incident.Title,
		incident.ServiceSummary,
		incident.ServiceID,
		incident.Status,
		incident.HTMLURL,
		incident.CreatedAt,
		incident.UpdatedAt,
		incident.AlertCount,
		incident.Urgency,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert incident %s: %w", incident.IncidentID, err)
	}

	return nil
}

// NEW METHOD - BatchUpsertIncidents performs batch upsert operations
func (db *DB) BatchUpsertIncidents(incidents []IncidentData) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		REPLACE INTO incidents (
			incident_id, incident_number, title, service_summary, 
			service_id, status, html_url, created_at, updated_at, 
			alert_count, urgency
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, incident := range incidents {
		_, err := stmt.Exec(
			incident.IncidentID,
			incident.IncidentNumber,
			incident.Title,
			incident.ServiceSummary,
			incident.ServiceID,
			incident.Status,
			incident.HTMLURL,
			incident.CreatedAt,
			incident.UpdatedAt,
			incident.AlertCount,
			incident.Urgency,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert incident %s: %w", incident.IncidentID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetOpenIncidents - ENHANCED WITH THREAD SAFETY AND ORDERING, SIGNATURE UNCHANGED
func (db *DB) GetOpenIncidents() ([]IncidentData, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT incident_id, incident_number, title, service_summary, 
			   service_id, status, html_url, created_at, updated_at, alert_count,
			   COALESCE(urgency, 'low') as urgency
		FROM incidents
		WHERE status IN ('triggered', 'acknowledged')
		ORDER BY 
			CASE status 
				WHEN 'triggered' THEN 1 
				WHEN 'acknowledged' THEN 2 
			END,
			created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query open incidents: %w", err)
	}
	defer rows.Close()

	var incidents []IncidentData
	for rows.Next() {
		var i IncidentData
		err := rows.Scan(
			&i.IncidentID,
			&i.IncidentNumber,
			&i.Title,
			&i.ServiceSummary,
			&i.ServiceID,
			&i.Status,
			&i.HTMLURL,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.AlertCount,
			&i.Urgency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return incidents, nil
}

// GetResolvedIncidents - ENHANCED WITH THREAD SAFETY, SIGNATURE UNCHANGED
func (db *DB) GetResolvedIncidents() ([]IncidentData, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT incident_id, incident_number, title, service_summary, 
			   service_id, status, html_url, created_at, updated_at, alert_count,
			   COALESCE(urgency, 'low') as urgency
		FROM incidents
		WHERE status = 'resolved'
		ORDER BY updated_at DESC
		LIMIT 100
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query resolved incidents: %w", err)
	}
	defer rows.Close()

	var incidents []IncidentData
	for rows.Next() {
		var i IncidentData
		err := rows.Scan(
			&i.IncidentID,
			&i.IncidentNumber,
			&i.Title,
			&i.ServiceSummary,
			&i.ServiceID,
			&i.Status,
			&i.HTMLURL,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.AlertCount,
			&i.Urgency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return incidents, nil
}

// ClearIncidents - ENHANCED WITH THREAD SAFETY, SIGNATURE UNCHANGED
func (db *DB) ClearIncidents() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `DELETE FROM incidents`
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear incidents: %w", err)
	}

	return nil
}

// GetResolvedIncidentsByServices - ENHANCED WITH THREAD SAFETY, SIGNATURE UNCHANGED
func (db *DB) GetResolvedIncidentsByServices(serviceIDs []string) ([]IncidentData, error) {
	if len(serviceIDs) == 0 {
		return []IncidentData{}, nil
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	// Build parameterized query with proper escaping
	args := make([]interface{}, len(serviceIDs))
	placeholders := make([]string, len(serviceIDs))
	for i, id := range serviceIDs {
		args[i] = id
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT incident_id, incident_number, title, service_summary, 
			   service_id, status, html_url, created_at, updated_at, alert_count,
			   COALESCE(urgency, 'low') as urgency
		FROM incidents
		WHERE status = 'resolved' AND service_id IN (%s)
		ORDER BY updated_at DESC
		LIMIT 100
	`, strings.Join(placeholders, ","))

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query resolved incidents by services: %w", err)
	}
	defer rows.Close()

	var incidents []IncidentData
	for rows.Next() {
		var i IncidentData
		err := rows.Scan(
			&i.IncidentID,
			&i.IncidentNumber,
			&i.Title,
			&i.ServiceSummary,
			&i.ServiceID,
			&i.Status,
			&i.HTMLURL,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.AlertCount,
			&i.Urgency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return incidents, nil
}

// NEW METHOD - GetIncidentStats returns statistics about incidents
func (db *DB) GetIncidentStats() (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count by status
	var triggered, acknowledged, resolved int
	err := db.conn.QueryRow(`
		SELECT 
			COUNT(CASE WHEN status = 'triggered' THEN 1 END) as triggered,
			COUNT(CASE WHEN status = 'acknowledged' THEN 1 END) as acknowledged,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved
		FROM incidents
	`).Scan(&triggered, &acknowledged, &resolved)

	if err != nil {
		return nil, fmt.Errorf("failed to get incident stats: %w", err)
	}

	stats["triggered"] = triggered
	stats["acknowledged"] = acknowledged
	stats["resolved"] = resolved
	stats["total"] = triggered + acknowledged + resolved

	return stats, nil
}
func (db *DB) GetNewestResolvedIncidentDate() (time.Time, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var updatedAt time.Time
	query := `
		SELECT updated_at
		FROM incidents
		WHERE status = 'resolved'
		ORDER BY updated_at DESC
		LIMIT 1
	`

	err := db.conn.QueryRow(query).Scan(&updatedAt)
	if err == sql.ErrNoRows {
		return time.Time{}, nil // No resolved incidents found
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get newest resolved incident date: %w", err)
	}

	return updatedAt, nil
}

func (db *DB) RemoveStaleOpenIncidents(currentIncidentIDs []string, serviceIDs []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(currentIncidentIDs) == 0 && len(serviceIDs) > 0 {
		// If no incidents returned from API but we have services, remove all open incidents for those services
		query := `
			UPDATE incidents 
			SET status = 'resolved', updated_at = CURRENT_TIMESTAMP
			WHERE status IN ('triggered', 'acknowledged')
		`

		if len(serviceIDs) > 0 {
			placeholders := make([]string, len(serviceIDs))
			args := make([]interface{}, len(serviceIDs))
			for i, id := range serviceIDs {
				placeholders[i] = "?"
				args[i] = id
			}
			query += fmt.Sprintf(" AND service_id IN (%s)", strings.Join(placeholders, ","))

			_, err := db.conn.Exec(query, args...)
			if err != nil {
				return fmt.Errorf("failed to remove all stale open incidents: %w", err)
			}
		}
		return nil
	}

	// Build NOT IN clause for incident IDs
	placeholders := make([]string, len(currentIncidentIDs))
	args := make([]interface{}, 0, len(currentIncidentIDs)+len(serviceIDs))

	for i, id := range currentIncidentIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE incidents 
		SET status = 'resolved', updated_at = CURRENT_TIMESTAMP
		WHERE status IN ('triggered', 'acknowledged')
		AND incident_id NOT IN (%s)
	`, strings.Join(placeholders, ","))

	// Add service filter if provided
	if len(serviceIDs) > 0 {
		servicePlaceholders := make([]string, len(serviceIDs))
		for i, id := range serviceIDs {
			servicePlaceholders[i] = "?"
			args = append(args, id)
		}
		query += fmt.Sprintf(" AND service_id IN (%s)", strings.Join(servicePlaceholders, ","))
	}

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove stale open incidents: %w", err)
	}

	return nil
}

func (db *DB) UpdateIncidentsBatch(incidents []IncidentData, staleIDs []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare upsert statement
	upsertStmt, err := tx.Prepare(`
		REPLACE INTO incidents (
			incident_id, incident_number, title, service_summary, 
			service_id, status, html_url, created_at, updated_at, 
			alert_count, urgency
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare upsert statement: %w", err)
	}
	defer upsertStmt.Close()

	// Upsert all current incidents
	for _, incident := range incidents {
		_, err := upsertStmt.Exec(
			incident.IncidentID,
			incident.IncidentNumber,
			incident.Title,
			incident.ServiceSummary,
			incident.ServiceID,
			incident.Status,
			incident.HTMLURL,
			incident.CreatedAt,
			incident.UpdatedAt,
			incident.AlertCount,
			incident.Urgency,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert incident %s: %w", incident.IncidentID, err)
		}
	}

	// Mark stale incidents as resolved
	if len(staleIDs) > 0 {
		placeholders := make([]string, len(staleIDs))
		args := make([]interface{}, len(staleIDs))
		for i, id := range staleIDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			UPDATE incidents 
			SET status = 'resolved', updated_at = CURRENT_TIMESTAMP
			WHERE incident_id IN (%s)
		`, strings.Join(placeholders, ","))

		_, err = tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to mark stale incidents as resolved: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetIncidentByID retrieves a single incident by its ID
func (db *DB) GetIncidentByID(incidentID string) (IncidentData, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT incident_id, incident_number, title, service_summary, 
			   service_id, status, html_url, created_at, updated_at, alert_count,
			   COALESCE(urgency, 'low') as urgency
		FROM incidents
		WHERE incident_id = ?
	`

	var incident IncidentData
	err := db.conn.QueryRow(query, incidentID).Scan(
		&incident.IncidentID,
		&incident.IncidentNumber,
		&incident.Title,
		&incident.ServiceSummary,
		&incident.ServiceID,
		&incident.Status,
		&incident.HTMLURL,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.AlertCount,
		&incident.Urgency,
	)

	if err == sql.ErrNoRows {
		return incident, fmt.Errorf("incident not found: %s", incidentID)
	}

	if err != nil {
		return incident, fmt.Errorf("failed to get incident: %w", err)
	}

	return incident, nil
}

// ClearIncidentSidebarCache removes cached alerts and notes for an incident
func (db *DB) ClearIncidentSidebarCache(incidentID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete alerts for this incident
	_, err = tx.Exec("DELETE FROM incident_alerts WHERE incident_id = ?", incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete alerts: %w", err)
	}

	// Delete notes for this incident
	_, err = tx.Exec("DELETE FROM incident_notes WHERE incident_id = ?", incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete notes: %w", err)
	}

	// Delete metadata for this incident
	_, err = tx.Exec("DELETE FROM incident_sidebar_metadata WHERE incident_id = ?", incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close - ORIGINAL METHOD UNCHANGED
func (db *DB) Close() error {
	return db.conn.Close()
}
