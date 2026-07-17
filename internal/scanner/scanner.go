package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

type Scanner struct {
	Workspace string
}

func NewScanner(workspace string) *Scanner {
	return &Scanner{Workspace: workspace}
}

type JournalEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type SearchResult struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

var ErrInvalidEntryPath = errors.New("invalid entry path")

func (s *Scanner) Get(entryPath string) (string, error) {
	if s.Workspace == "" || !strings.HasPrefix(entryPath, "/") || strings.Contains(entryPath, `\`) {
		return "", ErrInvalidEntryPath
	}
	relativePath := strings.TrimPrefix(entryPath, "/")
	if !fs.ValidPath(relativePath) || !strings.HasSuffix(strings.ToLower(relativePath), ".md") {
		return "", ErrInvalidEntryPath
	}
	content, err := s.readPath(entryPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s *Scanner) readPath(entryPath string) ([]byte, error) {
	relativePath := strings.TrimPrefix(entryPath, "/")
	if !fs.ValidPath(relativePath) || strings.Contains(relativePath, `\`) {
		return nil, ErrInvalidEntryPath
	}
	root, err := os.OpenRoot(s.Workspace)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(relativePath)
}

func (s *Scanner) markdownPaths() []string {
	if s.Workspace == "" {
		return []string{}
	}

	var paths []string
	filepath.WalkDir(s.Workspace, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(s.Workspace, path)
		if err == nil {
			paths = append(paths, "/"+filepath.ToSlash(rel))
		}
		return nil
	})

	sort.Slice(paths, func(i, j int) bool {
		baseI := filepath.Base(paths[i])
		baseJ := filepath.Base(paths[j])
		if baseI != baseJ {
			return baseI > baseJ
		}
		return paths[i] > paths[j]
	})
	return paths
}

func entryMonth(path string) string {
	date, err := time.Parse("2006-01-02", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if err != nil {
		return ""
	}
	return date.Format("2006-01")
}

func (s *Scanner) ListByMonth(page, limit int, month string) ([]JournalEntry, bool) {
	paths := s.markdownPaths()
	if month != "" {
		filtered := paths[:0]
		for _, path := range paths {
			if entryMonth(path) == month {
				filtered = append(filtered, path)
			}
		}
		paths = filtered
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 15
	}
	start := (page - 1) * limit
	if start >= len(paths) {
		return []JournalEntry{}, false
	}
	end := min(start+limit, len(paths))

	results := make([]JournalEntry, 0, end-start)
	for _, path := range paths[start:end] {
		content, err := s.readPath(path)
		if err == nil {
			results = append(results, JournalEntry{Path: path, Content: string(content)})
		}
	}
	return results, end < len(paths)
}

func (s *Scanner) Months() []string {
	seen := make(map[string]bool)
	var months []string
	for _, path := range s.markdownPaths() {
		month := entryMonth(path)
		if month != "" && !seen[month] {
			seen[month] = true
			months = append(months, month)
		}
	}
	return months
}

func matchingSnippet(line, query string) string {
	const maxRunes = 200
	text := []rune(strings.TrimSpace(line))
	if len(text) <= maxRunes {
		return string(text)
	}
	lowerText := []rune(strings.ToLower(string(text)))
	lowerQuery := []rune(strings.ToLower(query))
	match := 0
	for i := 0; i+len(lowerQuery) <= len(lowerText); i++ {
		if slices.Equal(lowerText[i:i+len(lowerQuery)], lowerQuery) {
			match = i
			break
		}
	}
	start := max(0, match-80)
	end := min(len(text), start+maxRunes)
	if end-start < maxRunes {
		start = max(0, end-maxRunes)
	}
	snippet := string(text[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(text) {
		snippet += "…"
	}
	return snippet
}

// Search scans file contents for a query string by walking the directory.
func (s *Scanner) Search(query string) []SearchResult {
	if query == "" || s.Workspace == "" {
		return []SearchResult{}
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult
	for _, path := range s.markdownPaths() {
		relPath := strings.TrimPrefix(path, "/")
		content, err := s.readPath(path)
		if err != nil {
			continue
		}

		text := string(content)
		if !strings.Contains(strings.ToLower(text), queryLower) {
			continue
		}

		defaultTitle := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
		title := defaultTitle
		snippet := ""
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") && title == defaultTitle {
				title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			}
			if snippet == "" && strings.Contains(strings.ToLower(trimmed), queryLower) {
				snippet = matchingSnippet(trimmed, query)
			}
		}
		results = append(results, SearchResult{Path: path, Title: title, Snippet: snippet})
	}
	return results
}
