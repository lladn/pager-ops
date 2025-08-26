package database

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// IncidentData represents an incident record in the database
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
}

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) createTables() error {
	query := `
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
		alert_count INTEGER
	)`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) UpsertIncident(incident IncidentData) error {
	query := `
	INSERT INTO incidents (incident_id, incident_number, title, service_summary, 
		service_id, status, html_url, created_at, updated_at, alert_count)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(incident_id) DO UPDATE SET
		incident_number = excluded.incident_number,
		title = excluded.title,
		service_summary = excluded.service_summary,
		service_id = excluded.service_id,
		status = excluded.status,
		html_url = excluded.html_url,
		created_at = excluded.created_at,
		updated_at = excluded.updated_at,
		alert_count = excluded.alert_count
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
	)
	return err
}

func (db *DB) GetOpenIncidents() ([]IncidentData, error) {
	query := `
	SELECT incident_id, incident_number, title, service_summary, 
		   service_id, status, html_url, created_at, updated_at, alert_count
	FROM incidents
	WHERE status IN ('triggered', 'acknowledged')
	ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
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
		)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, i)
	}

	return incidents, nil
}

func (db *DB) GetResolvedIncidents() ([]IncidentData, error) {
	query := `
	SELECT incident_id, incident_number, title, service_summary, 
		   service_id, status, html_url, created_at, updated_at, alert_count
	FROM incidents
	WHERE status = 'resolved'
	ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
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
		)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, i)
	}

	return incidents, nil
}

// ClearIncidents removes all incidents from the database
func (db *DB) ClearIncidents() error {
	query := `DELETE FROM incidents`
	_, err := db.conn.Exec(query)
	return err
}

// GetResolvedIncidentsByServices returns resolved incidents filtered by service IDs
func (db *DB) GetResolvedIncidentsByServices(serviceIDs []string) ([]IncidentData, error) {
	if len(serviceIDs) == 0 {
		return []IncidentData{}, nil
	}

	// Build the IN clause for service IDs
	args := make([]interface{}, len(serviceIDs))
	placeholders := make([]string, len(serviceIDs))
	for i, id := range serviceIDs {
		args[i] = id
		placeholders[i] = "?"
	}
	inClause := "(" + strings.Join(placeholders, ",") + ")"

	query := `
	SELECT incident_id, incident_number, title, service_summary, 
		   service_id, status, html_url, created_at, updated_at, alert_count
	FROM incidents
	WHERE status = 'resolved' AND service_id IN ` + inClause + `
	ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
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
		)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, i)
	}

	return incidents, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
