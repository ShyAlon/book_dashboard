package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

type Parsed struct {
	Title       string
	SourcePath  string
	SourceBytes []byte
	Text        string
}

func ParseFile(path string) (*Parsed, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var text string
	switch ext {
	case ".docx":
		text, err = parseDOCX(raw)
		if err != nil {
			return nil, err
		}
	case ".pdf":
		text, err = parsePDF(path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return &Parsed{
		Title:       title,
		SourcePath:  path,
		SourceBytes: raw,
		Text:        normalizeWhitespace(text),
	}, nil
}

func parseDOCX(raw []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}

	var xmlData []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, openErr := f.Open()
			if openErr != nil {
				return "", fmt.Errorf("open document.xml: %w", openErr)
			}
			defer rc.Close()
			xmlData, err = io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("read document.xml: %w", err)
			}
			break
		}
	}
	if len(xmlData) == 0 {
		return "", fmt.Errorf("word/document.xml not found")
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	var b strings.Builder
	inText := false
	for {
		tok, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return "", fmt.Errorf("decode document.xml: %w", tokenErr)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			}
			if t.Name.Local == "p" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inText {
				b.WriteString(string(t))
			}
		}
	}
	return b.String(), nil
}

func parsePDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var b strings.Builder
	total := r.NumPage()
	for i := 1; i <= total; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		content, pageErr := p.GetPlainText(nil)
		if pageErr != nil {
			continue
		}
		b.WriteString(content)
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no extractable text found in pdf")
	}
	return b.String(), nil
}

func normalizeWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.Join(strings.Fields(line), " ")
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
