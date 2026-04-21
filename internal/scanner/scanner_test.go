package scanner

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestScanner(t *testing.T) {
	// Setup temporary workspace
	tempDir, err := os.MkdirTemp("", "dailyflow_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := map[string]string{
		"2026-04-21.md": "Hello world today",
		"2026-04-20.md": "Yesterday was sunny",
		"notes/idea.md": "Greate idea content",
		"not_md.txt":    "Should ignore this",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	s := NewScanner(tempDir)

	// Test List
	list := s.List(1, 10)
	if len(list) != 3 {
		t.Errorf("Expected 3 markdown files, got %d", len(list))
	}
	if list[0].Content == "" {
		t.Errorf("Expected content to be populated in List")
	}

	// Test Search
	results, _ := s.Search("sunny")
	if len(results) != 1 || results[0].Path != "/2026-04-20.md" {
		t.Errorf("Search failed to find 'sunny' in correct file, got %v", results)
	}
	if results[0].Content != "Yesterday was sunny" {
		t.Errorf("Search results missing content")
	}

	results, _ = s.Search("idea")
	if len(results) != 1 || results[0].Path != "/notes/idea.md" {
		t.Errorf("Search failed for nested file")
	}
}

func TestSorting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dailyflow_sort_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	files := map[string]string{
		"2026/2026-03-31.md":    "march",
		"2026-04/2026-04-03.md": "april",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	s := NewScanner(tempDir)
	list := s.List(1, 10)

	if len(list) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(list))
	}

	// Expected newest first: 2026-04-03 then 2026-03-31
	if list[0].Path != "/2026-04/2026-04-03.md" {
		t.Errorf("Expected first file to be April, got %s", list[0].Path)
	}
	if list[1].Path != "/2026/2026-03-31.md" {
		t.Errorf("Expected second file to be March, got %s", list[1].Path)
	}

	// Test Search sorting
	results, _ := s.Search("r")
	if len(results) != 2 {
		t.Fatalf("Expected 2 search results, got %d", len(results))
	}
	if results[0].Path != "/2026-04/2026-04-03.md" {
		t.Errorf("Search: expected first result to be April, got %s", results[0].Path)
	}
}

func TestScanner_EmptyWorkspace(t *testing.T) {
	s := NewScanner("")
	if len(s.List(1, 10)) != 0 {
		t.Error("List should return empty slice for empty workspace")
	}
	res, _ := s.Search("test")
	if len(res) != 0 {
		t.Error("Search should return empty slice for empty workspace")
	}
}

func TestScanner_Pagination(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "dailyflow_page_test")
	defer os.RemoveAll(tempDir)

	for i := 1; i <= 10; i++ {
		name := filepath.Join(tempDir, "file"+string(rune('a'+i))+".md")
		os.WriteFile(name, []byte("content"), 0644)
	}

	s := NewScanner(tempDir)

	// Test page < 1
	if len(s.List(0, 5)) != 5 {
		t.Error("Should default to page 1 if page < 1")
	}

	// Test limit <= 0
	if len(s.List(1, 0)) != 10 {
		t.Errorf("Expected 10 files with default limit, got %d", len(s.List(1, 0)))
	}

	// Test specific page/limit
	list := s.List(2, 3)
	if len(list) != 3 {
		t.Errorf("Expected 3 files on page 2, got %d", len(list))
	}

	// Test out of bounds
	if len(s.List(10, 10)) != 0 {
		t.Error("Should return empty for page out of range")
	}
}

func TestScanner_Search_EmptyQuery(t *testing.T) {
	s := NewScanner("some/path")
	res, _ := s.Search("")
	if len(res) != 0 {
		t.Error("Empty query should return empty results")
	}
}

func TestScanner_NoMarkdownFiles(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "dailyflow_nomd_test")
	defer os.RemoveAll(tempDir)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("text"), 0644)

	s := NewScanner(tempDir)
	if len(s.List(1, 10)) != 0 {
		t.Error("Expected 0 results for dir with no md files")
	}
}

func TestScanner_ConcurrentSearch(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "dailyflow_concurrent_test")
	defer os.RemoveAll(tempDir)

	count := 50
	for i := 0; i < count; i++ {
		name := filepath.Join(tempDir, "file"+strconv.Itoa(i)+".md")
		content := "searchable content"
		if i%2 == 0 {
			content = "other"
		}
		os.WriteFile(name, []byte(content), 0644)
	}

	s := NewScanner(tempDir)
	results, err := s.Search("searchable")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != count/2 {
		t.Errorf("Expected %d results, got %d", count/2, len(results))
	}
}
