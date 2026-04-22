package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"codeberg.org/go-pdf/fpdf"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// section represents a group of files. Files within a section that has
// continuous=true are concatenated without page breaks between them.
type section struct {
	files      []string
	continuous bool
}

func run() error {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	sections, err := collectSections(root)
	if err != nil {
		return fmt.Errorf("collecting files: %w", err)
	}

	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 15)

	totalFiles := 0
	for _, sec := range sections {
		for i, f := range sec.files {
			content, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				return fmt.Errorf("reading %s: %w", f, err)
			}

			if i == 0 || !sec.continuous {
				pdf.AddPage()
			} else {
				// Whitespace for notes + separator between entries
				pdf.Ln(75)
				pdf.SetDrawColor(180, 180, 180)
				pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
				pdf.Ln(5)
			}

			// File path header
			pdf.SetFont("Courier", "B", 10)
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(0, 5, f, "", 1, "", false, 0, "")
			pdf.Ln(3)

			// Horizontal rule
			pdf.SetDrawColor(200, 200, 200)
			pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
			pdf.Ln(3)

			// File content
			pdf.SetFont("Courier", "", 8)
			pdf.SetTextColor(0, 0, 0)
			pdf.MultiCell(0, 3.5, string(content), "", "", false)

			totalFiles++
		}
	}

	output := filepath.Join(root, "printout.pdf")
	if err := pdf.OutputFileAndClose(output); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Printf("Wrote %s (%d files)\n", output, totalFiles)
	return nil
}

func collectSections(root string) ([]section, error) {
	var sections []section

	// 1. Top-level README
	sections = append(sections, section{files: []string{"README.md"}})

	// 2. All mistakes (directories + standalone) in one continuous section
	var mistakeFiles []string
	dirFiles, err := collectMistakeDirectories(root)
	if err != nil {
		return nil, err
	}
	mistakeFiles = append(mistakeFiles, dirFiles...)
	standaloneFiles, err := collectStandaloneMarkdown(root)
	if err != nil {
		return nil, err
	}
	mistakeFiles = append(mistakeFiles, standaloneFiles...)
	if len(mistakeFiles) > 0 {
		sections = append(sections, section{files: mistakeFiles, continuous: true})
	}

	// 3. Glossary terms (concatenated together, no page breaks between them)
	termFiles, err := collectTerms(root)
	if err != nil {
		return nil, err
	}
	if len(termFiles) > 0 {
		sections = append(sections, section{files: termFiles, continuous: true})
	}

	return sections, nil
}

func collectMistakeDirectories(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "src"))
	if err != nil {
		return nil, fmt.Errorf("reading src/: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "terms" {
			continue
		}

		dirPath := filepath.Join("src", entry.Name())
		dirEntries, err := os.ReadDir(filepath.Join(root, dirPath))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", dirPath, err)
		}

		// Collect all files in this mistake directory, README first then sorted
		var dirFiles []string
		for _, f := range dirEntries {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if strings.HasSuffix(name, ".md") {
				dirFiles = append(dirFiles, name)
			}
		}

		// Sort with README.md first, then alphabetically
		sort.Slice(dirFiles, func(i, j int) bool {
			if dirFiles[i] == "README.md" {
				return true
			}
			if dirFiles[j] == "README.md" {
				return false
			}
			return dirFiles[i] < dirFiles[j]
		})

		for _, f := range dirFiles {
			files = append(files, filepath.Join(dirPath, f))
		}
	}

	return files, nil
}

func collectStandaloneMarkdown(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "src"))
	if err != nil {
		return nil, fmt.Errorf("reading src/: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join("src", entry.Name()))
		}
	}

	slices.Sort(files)
	return files, nil
}

func collectTerms(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "src", "terms"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading src/terms/: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join("src", "terms", entry.Name()))
		}
	}

	slices.Sort(files)
	return files, nil
}
