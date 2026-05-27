package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type Identity struct {
	ID        int
	Name      string
	Email     string
	Phone     string
	Metadata  string // JSON blob
	Status    string
	CreatedAt string
}

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil { return nil, err }

	query := `
	CREATE TABLE IF NOT EXISTS identities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT,
		metadata TEXT,
		status TEXT DEFAULT 'forged',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil { return nil, err }
	return &Store{db: db}, nil
}

func (s *Store) SaveIdentity(name, email, phone, metadata string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO identities (name, email, phone, metadata) VALUES (?, ?, ?, ?)", name, email, phone, metadata)
	if err != nil { return 0, err }
	return res.LastInsertId()
}

func (s *Store) GetIdentity(id int) (*Identity, error) {
	var i Identity
	err := s.db.QueryRow("SELECT id, name, email, phone, metadata, status, created_at FROM identities WHERE id = ?", id).
		Scan(&i.ID, &i.Name, &i.Email, &i.Phone, &i.Metadata, &i.Status, &i.CreatedAt)
	if err != nil { return nil, err }
	return &i, nil
}

func (s *Store) ListIdentities() ([]Identity, error) {
	rows, err := s.db.Query("SELECT id, name, status FROM identities")
	if err != nil { return nil, err }
	defer rows.Close()

	var list []Identity
	for rows.Next() {
		var i Identity
		rows.Scan(&i.ID, &i.Name, &i.Status)
		list = append(list, i)
	}
	return list, nil
}
