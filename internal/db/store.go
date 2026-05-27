package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "modernc.org/sqlite"
)

// ---- Models ----

type List struct {
	ID          int
	Name        string
	CadenceDays int // minimum days between sends to the same subscriber
	CreatedAt   string
}

type Subscriber struct {
	ID     int
	ListID int
	Email  string
	Name   string
	Status string // pending | confirmed | unsubscribed
}

type Message struct {
	ID         int
	ThreadID   string
	Direction  string // in | out
	FromAddr   string
	ToAddr     string
	Subject    string
	Body       string
	Status     string // received | drafted | sent | escalated
	Category   string
	Confidence float64
	MessageID  string
	CreatedAt  string
}

type Campaign struct {
	ID        int
	ListID    int
	Name      string
	Persona   string
	Subject   string
	Body      string
	Status    string // draft | sending | sent
	SentCount int
	CreatedAt string
}

type Store struct{ db *sql.DB }

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS lists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		cadence_days INTEGER NOT NULL DEFAULT 4,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS subscribers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id INTEGER NOT NULL,
		email TEXT NOT NULL,
		name TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		confirm_token TEXT,
		consent_source TEXT,
		confirmed_at DATETIME,
		last_sent_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(list_id, email)
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		thread_id TEXT,
		direction TEXT NOT NULL,
		from_addr TEXT,
		to_addr TEXT,
		subject TEXT,
		body TEXT,
		status TEXT NOT NULL,
		category TEXT,
		confidence REAL,
		message_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS campaigns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		persona TEXT,
		subject TEXT,
		body TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		sent_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS suppression (
		email TEXT PRIMARY KEY,
		reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- Lists ----

func (s *Store) CreateList(name string, cadenceDays int) (int64, error) {
	res, err := s.db.Exec("INSERT INTO lists (name, cadence_days) VALUES (?, ?)", name, cadenceDays)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetList(id int) (*List, error) {
	var l List
	err := s.db.QueryRow("SELECT id, name, cadence_days, created_at FROM lists WHERE id = ?", id).
		Scan(&l.ID, &l.Name, &l.CadenceDays, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Store) ListLists() ([]List, error) {
	rows, err := s.db.Query("SELECT id, name, cadence_days, created_at FROM lists ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []List
	for rows.Next() {
		var l List
		if err := rows.Scan(&l.ID, &l.Name, &l.CadenceDays, &l.CreatedAt); err != nil {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

// ---- Subscribers (double opt-in) ----

// AddSubscriber inserts a pending subscriber and returns its confirmation token.
// The subscriber is not mailable until ConfirmSubscriber is called with the token.
func (s *Store) AddSubscriber(listID int, email, name, source string) (string, error) {
	token := newToken()
	_, err := s.db.Exec(
		"INSERT INTO subscribers (list_id, email, name, status, confirm_token, consent_source) VALUES (?, ?, ?, 'pending', ?, ?)",
		listID, email, name, token, source)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ConfirmSubscriber promotes a pending subscriber to confirmed via its token.
func (s *Store) ConfirmSubscriber(token string) (*Subscriber, error) {
	var sub Subscriber
	err := s.db.QueryRow(
		"SELECT id, list_id, email, name FROM subscribers WHERE confirm_token = ? AND status = 'pending'", token).
		Scan(&sub.ID, &sub.ListID, &sub.Email, &sub.Name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or already-used confirmation token")
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(
		"UPDATE subscribers SET status = 'confirmed', confirmed_at = CURRENT_TIMESTAMP, confirm_token = NULL WHERE id = ?", sub.ID); err != nil {
		return nil, err
	}
	sub.Status = "confirmed"
	return &sub, nil
}

// Unsubscribe marks every list membership for an email as unsubscribed and adds
// the address to the suppression list (the hard send-time gate).
func (s *Store) Unsubscribe(email, reason string) error {
	if _, err := s.db.Exec("UPDATE subscribers SET status = 'unsubscribed' WHERE email = ?", email); err != nil {
		return err
	}
	return s.Suppress(email, reason)
}

// DueSubscribers returns confirmed, non-suppressed subscribers of a list whose
// last send is older than the cadence window (or who have never been sent to).
func (s *Store) DueSubscribers(listID, cadenceDays int) ([]Subscriber, error) {
	q := `SELECT id, list_id, email, name FROM subscribers
		WHERE list_id = ? AND status = 'confirmed'
		  AND email NOT IN (SELECT email FROM suppression)
		  AND (last_sent_at IS NULL OR last_sent_at <= datetime('now', ?))`
	rows, err := s.db.Query(q, listID, fmt.Sprintf("-%d days", cadenceDays))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.ListID, &sub.Email, &sub.Name); err != nil {
			continue
		}
		sub.Status = "confirmed"
		out = append(out, sub)
	}
	return out, nil
}

// MarkSent records that a subscriber was just sent a campaign (cadence anchor).
func (s *Store) MarkSent(subscriberID int) error {
	_, err := s.db.Exec("UPDATE subscribers SET last_sent_at = CURRENT_TIMESTAMP WHERE id = ?", subscriberID)
	return err
}

// ListSubscribers returns every subscriber of a list, any status, ordered by id.
func (s *Store) ListSubscribers(listID int) ([]Subscriber, error) {
	rows, err := s.db.Query("SELECT id, list_id, email, name, status FROM subscribers WHERE list_id = ? ORDER BY id", listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.ListID, &sub.Email, &sub.Name, &sub.Status); err != nil {
			continue
		}
		out = append(out, sub)
	}
	return out, nil
}

// ---- Suppression ----

func (s *Store) Suppress(email, reason string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO suppression (email, reason) VALUES (?, ?)", email, reason)
	return err
}

func (s *Store) IsSuppressed(email string) (bool, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(1) FROM suppression WHERE email = ?", email).Scan(&n)
	return n > 0, err
}

// ---- Messages ----

func (s *Store) SaveMessage(m Message) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO messages (thread_id, direction, from_addr, to_addr, subject, body, status, category, confidence, message_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ThreadID, m.Direction, m.FromAddr, m.ToAddr, m.Subject, m.Body, m.Status, m.Category, m.Confidence, m.MessageID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) MessagesByStatus(status string) ([]Message, error) {
	rows, err := s.db.Query(
		"SELECT id, thread_id, direction, from_addr, to_addr, subject, body, status, category, confidence, message_id, created_at FROM messages WHERE status = ? ORDER BY id",
		status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Direction, &m.FromAddr, &m.ToAddr, &m.Subject, &m.Body, &m.Status, &m.Category, &m.Confidence, &m.MessageID, &m.CreatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *Store) UpdateMessageStatus(id int, status string) error {
	_, err := s.db.Exec("UPDATE messages SET status = ? WHERE id = ?", status, id)
	return err
}

// ---- Campaigns ----

func (s *Store) CreateCampaign(listID int, name, persona, subject, body string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO campaigns (list_id, name, persona, subject, body) VALUES (?, ?, ?, ?, ?)",
		listID, name, persona, subject, body)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateCampaignStatus(id int, status string, sentCount int) error {
	_, err := s.db.Exec("UPDATE campaigns SET status = ?, sent_count = ? WHERE id = ?", status, sentCount, id)
	return err
}
