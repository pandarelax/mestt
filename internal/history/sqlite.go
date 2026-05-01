package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"pandarelax/mestt/internal/paths"
)

type Entry struct {
	ID         int64
	CreatedAt  time.Time
	SourceKind string
	SourcePath string
	ModelID    string
	Transcript string
}

type Store struct {
	db *sql.DB
}

func Open() (*Store, error) {
	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return nil, fmt.Errorf("ensure history directories: %w", err)
	}

	db, err := sql.Open("sqlite", p.HistoryDB)
	if err != nil {
		return nil, fmt.Errorf("open history database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Save(ctx context.Context, entry Entry) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO history (created_at, source_kind, source_path, model_id, transcript)
		VALUES (?, ?, ?, ?, ?)
	`, entry.CreatedAt.UTC().Format(time.RFC3339), entry.SourceKind, entry.SourcePath, entry.ModelID, entry.Transcript)
	if err != nil {
		return fmt.Errorf("insert history entry: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, source_kind, source_path, model_id, transcript
		FROM history
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query history entries: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		var createdAt string
		if err := rows.Scan(&entry.ID, &createdAt, &entry.SourceKind, &entry.SourcePath, &entry.ModelID, &entry.Transcript); err != nil {
			return nil, fmt.Errorf("scan history entry: %w", err)
		}
		entry.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse history timestamp: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history rows: %w", err)
	}

	return entries, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			source_kind TEXT NOT NULL,
			source_path TEXT,
			model_id TEXT NOT NULL,
			transcript TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate history schema: %w", err)
	}
	return nil
}
