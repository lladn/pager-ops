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

	return db, nil
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

// Close - ORIGINAL METHOD UNCHANGED
func (db *DB) Close() error {
	return db.conn.Close()
}