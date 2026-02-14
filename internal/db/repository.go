package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"book_dashboard/internal/forensics"
)

func PersistContradictions(dbPath string, contradictions []forensics.Contradiction) error {
	conn, err := Open(dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM contradictions`); err != nil {
		return fmt.Errorf("clear contradictions: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM entities`); err != nil {
		return fmt.Errorf("clear entities: %w", err)
	}

	entityIDs := map[string]int64{}
	for _, c := range contradictions {
		name := strings.TrimSpace(c.EntityName)
		if name == "" {
			name = "unknown"
		}
		id, ok := entityIDs[name]
		if !ok {
			aliases, _ := json.Marshal([]string{})
			attributes, _ := json.Marshal(map[string]string{})
			res, err := tx.Exec(`INSERT INTO entities(name, aliases, attributes) VALUES(?,?,?)`, name, string(aliases), string(attributes))
			if err != nil {
				return fmt.Errorf("insert entity: %w", err)
			}
			id, err = res.LastInsertId()
			if err != nil {
				return fmt.Errorf("entity last insert id: %w", err)
			}
			entityIDs[name] = id
		}

		if _, err := tx.Exec(
			`INSERT INTO contradictions(entity_id, chapter_a, chapter_b, description, severity) VALUES(?,?,?,?,?)`,
			id,
			c.ChapterA,
			c.ChapterB,
			c.Description,
			c.Severity,
		); err != nil {
			return fmt.Errorf("insert contradiction: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func CountRows(dbPath, table string) (int, error) {
	conn, err := Open(dbPath)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return countRowsConn(conn, table)
}

func countRowsConn(conn *sql.DB, table string) (int, error) {
	row := conn.QueryRow(`SELECT COUNT(*) FROM ` + table)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("scan count: %w", err)
	}
	return count, nil
}
