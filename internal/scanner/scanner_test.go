package scanner

import (
	"os"
	"path/filepath"
	"strings"
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
		"2026-04-20.md": "# Weather Note\nYesterday was sunny and warm.",
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
	if results[0].Title != "Weather Note" {
		t.Errorf("Expected title from Markdown heading, got %q", results[0].Title)
	}
	if results[0].Snippet != "Yesterday was sunny and warm." {
		t.Errorf("Expected matching line as snippet, got %q", results[0].Snippet)
	}

	results, _ = s.Search("idea")
	if len(results) != 1 || results[0].Path != "/notes/idea.md" {
		t.Errorf("Search failed for nested file")
	}
}

func TestGetEntryStaysInsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(workspace, "2026", "07", "2026-07-17.md")
	if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "escape.md")); err != nil {
		t.Fatal(err)
	}

	s := NewScanner(workspace)
	content, err := s.Get("/2026/07/2026-07-17.md")
	if err != nil || content != "inside" {
		t.Fatalf("Expected nested entry, got %q, %v", content, err)
	}
	for _, path := range []string{"", outside, "/../outside.md", "/2026\\07\\entry.md", "/image.png", "/escape.md"} {
		if _, err := s.Get(path); err == nil {
			t.Errorf("Expected unsafe entry path %q to fail", path)
		}
	}
	for _, entry := range s.List(1, 20) {
		if entry.Path == "/escape.md" {
			t.Fatal("Timeline must not follow symlinks outside the workspace")
		}
	}
	results, err := s.Search("outside")
	if err != nil || len(results) != 0 {
		t.Fatalf("Search must not follow symlinks outside the workspace: %v, %v", results, err)
	}
}

func TestSearchSnippetKeepsTheMatchInLongLines(t *testing.T) {
	tempDir := t.TempDir()
	content := strings.Repeat("before ", 80) + "NEEDLE" + strings.Repeat(" after", 80)
	if err := os.WriteFile(filepath.Join(tempDir, "2026-04-21.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := NewScanner(tempDir).Search("needle")
	if err != nil || len(results) != 1 {
		t.Fatalf("Unexpected search result: %v, %v", results, err)
	}
	if len([]rune(results[0].Snippet)) > 203 {
		t.Fatalf("Snippet is too long: %d", len([]rune(results[0].Snippet)))
	}
	if !strings.Contains(strings.ToLower(results[0].Snippet), "needle") {
		t.Fatalf("Snippet lost the match: %q", results[0].Snippet)
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

func TestMonthsAndMonthlyList(t *testing.T) {
	tempDir := t.TempDir()
	files := map[string]string{
		"2026/04/2026-04-21.md": "latest",
		"2026/04/2026-04-20.md": "earlier",
		"2026/03/2026-03-31.md": "march",
		"notes/idea.md":         "not a dated journal",
	}
	for name, content := range files {
		path := filepath.Join(tempDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := NewScanner(tempDir)
	months := s.Months()
	if len(months) != 2 || months[0] != "2026-04" || months[1] != "2026-03" {
		t.Fatalf("Unexpected months: %v", months)
	}
	april := s.ListByMonth(1, 15, "2026-04")
	if len(april) != 2 || april[0].Path != "/2026/04/2026-04-21.md" {
		t.Fatalf("Unexpected April entries: %v", april)
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
