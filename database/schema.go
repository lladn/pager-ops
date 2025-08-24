package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type IncidentData struct {
	IncidentID     string
	IncidentNumber int
	Title          string
	ServiceSummary string
	ServiceID      string
	Status         string
	HTMLURL        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AlertCount     int
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
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
	);
	
	CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
	CREATE INDEX IF NOT EXISTS idx_incidents_service_id ON incidents(service_id);
	CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents(created_at);
	`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) UpsertIncident(incident IncidentData) error {
	query := `
	INSERT INTO incidents (
		incident_id, incident_number, title, service_summary, 
		service_id, status, html_url, created_at, updated_at, alert_count
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

func (db *DB) Close() error {
	return db.conn.Close()
}
