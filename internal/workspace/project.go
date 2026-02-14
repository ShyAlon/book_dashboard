package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Report struct {
	BookTitle      string   `json:"book_title"`
	WordCount      int      `json:"word_count"`
	MHDScore       int      `json:"mhd_score"`
	Contradictions int      `json:"contradictions"`
	SlopFlags      []string `json:"slop_flags"`
	Analysis       any      `json:"analysis,omitempty"`
}

type ProjectInfo struct {
	ID         string
	Root       string
	SourcePath string
	ReportPath string
}

func CreateProject(workspaceRoot, bookTitle string, source []byte) (*ProjectInfo, error) {
	return CreateProjectWithSource(workspaceRoot, bookTitle, "source.docx", source)
}

func CreateProjectWithSource(workspaceRoot, bookTitle, sourceFileName string, source []byte) (*ProjectInfo, error) {
	id := bookTitleHash(bookTitle)
	projectRoot := filepath.Join(workspaceRoot, "projects", id)
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create project dir: %w", err)
	}

	sourceFileName = sanitizeSourceName(sourceFileName)
	sourcePath := filepath.Join(projectRoot, sourceFileName)
	if len(source) > 0 {
		if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
			return nil, fmt.Errorf("write source file: %w", err)
		}
	} else if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		if err := os.WriteFile(sourcePath, nil, 0o644); err != nil {
			return nil, fmt.Errorf("create empty source file: %w", err)
		}
	}

	reportPath := filepath.Join(projectRoot, "report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		report := Report{
			BookTitle:      strings.TrimSpace(bookTitle),
			WordCount:      0,
			MHDScore:       0,
			Contradictions: 0,
			SlopFlags:      []string{},
		}
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal report: %w", err)
		}
		if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
			return nil, fmt.Errorf("write report: %w", err)
		}
	}

	return &ProjectInfo{
		ID:         id,
		Root:       projectRoot,
		SourcePath: sourcePath,
		ReportPath: reportPath,
	}, nil
}

func SaveReport(path string, report Report) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func bookTitleHash(title string) string {
	trimmed := strings.TrimSpace(strings.ToLower(title))
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:12]
}

func sanitizeSourceName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "source.docx"
	}
	return strings.ReplaceAll(base, "..", "")
}
