package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const SchemaSQL = `
CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY,
    name TEXT,
    aliases TEXT,
    attributes TEXT
);

CREATE TABLE IF NOT EXISTS contradictions (
    id INTEGER PRIMARY KEY,
    entity_id INTEGER,
    chapter_a INTEGER,
    chapter_b INTEGER,
    description TEXT,
    severity TEXT
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(SchemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}
