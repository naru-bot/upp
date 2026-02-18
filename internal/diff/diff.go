package diff

import (
	"fmt"
	"strings"
)

type Change struct {
	Type string `json:"type"` // added, removed, context
	Line string `json:"line"`
	Num  int    `json:"line_num"`
}

type DiffResult struct {
	HasChanges bool     `json:"has_changes"`
	Changes    []Change `json:"changes"`
	Summary    string   `json:"summary"`
	Added      int      `json:"lines_added"`
	Removed    int      `json:"lines_removed"`
}

// Diff computes a simple line-based diff between old and new content
func Diff(oldContent, newContent string) *DiffResult {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	result := &DiffResult{}

	if oldContent == newContent {
		result.Summary = "No changes"
		return result
	}

	result.HasChanges = true

	// Simple LCS-based diff
	lcs := lcsMatrix(oldLines, newLines)
	changes := backtrack(lcs, oldLines, newLines, len(oldLines), len(newLines))
	result.Changes = changes

	for _, c := range changes {
		switch c.Type {
		case "added":
			result.Added++
		case "removed":
			result.Removed++
		}
	}

	result.Summary = fmt.Sprintf("+%d lines, -%d lines", result.Added, result.Removed)
	return result
}

// FormatUnified returns a unified diff string
func FormatUnified(d *DiffResult, oldName, newName string) string {
	if !d.HasChanges {
		return "No changes detected.\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n+++ %s\n", oldName, newName))

	for _, c := range d.Changes {
		switch c.Type {
		case "removed":
			sb.WriteString(fmt.Sprintf("\033[31m- %s\033[0m\n", c.Line))
		case "added":
			sb.WriteString(fmt.Sprintf("\033[32m+ %s\033[0m\n", c.Line))
		case "context":
			sb.WriteString(fmt.Sprintf("  %s\n", c.Line))
		}
	}
	return sb.String()
}

// FormatPlain returns diff without color codes (for --json or piping)
func FormatPlain(d *DiffResult) string {
	if !d.HasChanges {
		return "No changes detected.\n"
	}

	var sb strings.Builder
	for _, c := range d.Changes {
		switch c.Type {
		case "removed":
			sb.WriteString(fmt.Sprintf("- %s\n", c.Line))
		case "added":
			sb.WriteString(fmt.Sprintf("+ %s\n", c.Line))
		case "context":
			sb.WriteString(fmt.Sprintf("  %s\n", c.Line))
		}
	}
	return sb.String()
}

func lcsMatrix(a, b []string) [][]int {
	m := len(a)
	n := len(b)
	matrix := make([][]int, m+1)
	for i := range matrix {
		matrix[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				matrix[i][j] = matrix[i-1][j-1] + 1
			} else if matrix[i-1][j] >= matrix[i][j-1] {
				matrix[i][j] = matrix[i-1][j]
			} else {
				matrix[i][j] = matrix[i][j-1]
			}
		}
	}
	return matrix
}

func backtrack(lcs [][]int, a, b []string, i, j int) []Change {
	if i == 0 && j == 0 {
		return nil
	}
	if i > 0 && j > 0 && a[i-1] == b[j-1] {
		changes := backtrack(lcs, a, b, i-1, j-1)
		return append(changes, Change{Type: "context", Line: a[i-1], Num: i})
	}
	if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
		changes := backtrack(lcs, a, b, i, j-1)
		return append(changes, Change{Type: "added", Line: b[j-1], Num: j})
	}
	changes := backtrack(lcs, a, b, i-1, j)
	return append(changes, Change{Type: "removed", Line: a[i-1], Num: i})
}
