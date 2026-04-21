package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

// List returns a paginated list of entries by scanning the directory.
func (s *Scanner) List(page, limit int) []JournalEntry {
	if s.Workspace == "" {
		return []JournalEntry{}
	}

	var files []string
	filepath.WalkDir(s.Workspace, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			rel, err := filepath.Rel(s.Workspace, path)
			if err != nil {
				return nil
			}
			relPath := filepath.ToSlash(rel)
			files = append(files, "/"+relPath)
		}
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		baseI := filepath.Base(files[i])
		baseJ := filepath.Base(files[j])
		if baseI != baseJ {
			return baseI > baseJ
		}
		return files[i] > files[j]
	})

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 15
	}

	start := (page - 1) * limit
	if start >= len(files) {
		return []JournalEntry{}
	}

	end := start + limit
	if end > len(files) {
		end = len(files)
	}

	selectedPaths := files[start:end]
	results := make([]JournalEntry, len(selectedPaths))
	for i, p := range selectedPaths {
		fullPath := filepath.Join(s.Workspace, p)
		content, _ := os.ReadFile(fullPath)
		results[i] = JournalEntry{
			Path:    p,
			Content: string(content),
		}
	}

	return results
}

// Search scans file contents for a query string by walking the directory.
func (s *Scanner) Search(query string) ([]JournalEntry, error) {
	if query == "" || s.Workspace == "" {
		return []JournalEntry{}, nil
	}

	var files []string
	filepath.WalkDir(s.Workspace, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			rel, err := filepath.Rel(s.Workspace, path)
			if err != nil {
				return nil
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})

	var results []JournalEntry
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, relPath := range files {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullPath := filepath.Join(s.Workspace, p)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return
			}

			if strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
				mu.Lock()
				results = append(results, JournalEntry{
					Path:    "/" + p,
					Content: string(content),
				})
				mu.Unlock()
			}
		}(relPath)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		baseI := filepath.Base(results[i].Path)
		baseJ := filepath.Base(results[j].Path)
		if baseI != baseJ {
			return baseI > baseJ
		}
		return results[i].Path > results[j].Path
	})

	return results, nil
}
