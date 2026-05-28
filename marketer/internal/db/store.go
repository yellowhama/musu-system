package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Campaign struct {
	ID        int
	Topic     string
	Content   string
	Brief     string // New: Strategic Brief
	Status    string
	Persona   string
	CreatedAt string
}

type Store struct {
	db *sql.DB
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func NewStore(dbPath string) (*Store, error) {
	// Ensure the parent directory exists so cwd-isolated invocations (notably
	// MCP servers) don't fail with SQLITE_CANTOPEN on a missing projects tree.
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil { return nil, err }

	// Migration logic: Check if 'brief' column exists, if not, add it.
	// For simplicity in this setup, we use CREATE TABLE with it, 
	// and execute an ALTER TABLE just in case.
	query := `
	CREATE TABLE IF NOT EXISTS campaigns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic TEXT NOT NULL,
		content TEXT NOT NULL,
		brief TEXT,
		status TEXT DEFAULT 'draft',
		persona TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil { return nil, err }
	
	// Ensure brief column exists in existing DBs
	db.Exec("ALTER TABLE campaigns ADD COLUMN brief TEXT") 

	return &Store{db: db}, nil
}

func (s *Store) SaveCampaign(topic, content, brief, persona string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO campaigns (topic, content, brief, persona) VALUES (?, ?, ?, ?)", topic, content, brief, persona)
	if err != nil { return 0, err }
	return res.LastInsertId()
}

func (s *Store) ListCampaigns() ([]Campaign, error) {
	rows, err := s.db.Query("SELECT id, topic, brief, status, persona, created_at FROM campaigns ORDER BY id DESC")
	if err != nil { return nil, err }
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		var brief sql.NullString
		if err := rows.Scan(&c.ID, &c.Topic, &brief, &c.Status, &c.Persona, &c.CreatedAt); err != nil { continue }
		c.Brief = brief.String
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

func (s *Store) GetCampaign(id int) (*Campaign, error) {
	var c Campaign
	var brief sql.NullString
	err := s.db.QueryRow("SELECT id, topic, content, brief, status, persona, created_at FROM campaigns WHERE id = ?", id).
		Scan(&c.ID, &c.Topic, &c.Content, &brief, &c.Status, &c.Persona, &c.CreatedAt)
	if err != nil { return nil, err }
	c.Brief = brief.String
	return &c, nil
}

func (s *Store) UpdateStatus(id int, status string) (int64, error) {
	res, err := s.db.Exec("UPDATE campaigns SET status = ? WHERE id = ?", status, id)
	if err != nil { return 0, err }
	return res.RowsAffected()
}

func (s *Store) GetRecentPublishedCampaigns(limit int) ([]Campaign, error) {
	rows, err := s.db.Query("SELECT topic, brief FROM campaigns WHERE status = 'published' ORDER BY id DESC LIMIT ?", limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var list []Campaign
	for rows.Next() {
		var c Campaign
		if err := rows.Scan(&c.Topic, &c.Brief); err != nil { continue }
		list = append(list, c)
	}
	return list, nil
}
