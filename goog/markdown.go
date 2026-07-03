package goog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MarkdownJob extends ImageJob with the source markdown file path.
type MarkdownJob struct {
	SourceFile string // path to the .md file
	ImageJob
}

// ogImageFrontmatter represents the ogImage block in YAML frontmatter.
type ogImageFrontmatter struct {
	Title       string `yaml:"title"`
	Tag         string `yaml:"tag"`
	Description string `yaml:"description"`
	SiteName    string `yaml:"site_name"`
	Template    string `yaml:"template"`
	Raw         bool   `yaml:"raw"`
}

// frontmatter wraps the top-level frontmatter structure.
type frontmatter struct {
	OGImage *ogImageFrontmatter `yaml:"ogImage"`
}

// ScanMarkdown walks a directory tree looking for .md files with ogImage
// frontmatter. Returns a list of MarkdownJobs ready for generation.
// Images are placed adjacent to their source files (e.g., post.md → post.png).
func ScanMarkdown(root string, ignorePatterns []string) ([]MarkdownJob, error) {
	var jobs []MarkdownJob

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Check ignore patterns
		relPath, _ := filepath.Rel(root, path)
		for _, pattern := range ignorePatterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			// Match against both relative path and filename
			matched, _ := filepath.Match(pattern, relPath)
			matchedBase, _ := filepath.Match(pattern, filepath.Base(path))
			if matched || matchedBase {
				return nil
			}
		}

		// Try to parse frontmatter
		fm, err := parseFrontmatter(path)
		if err != nil {
			// Skip files with parse errors
			return nil
		}

		if fm == nil || fm.OGImage == nil {
			// No ogImage frontmatter, skip
			return nil
		}

		og := fm.OGImage

		// Generate output path: same directory, same base name, .png extension
		dir := filepath.Dir(path)
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		outPath := filepath.Join(dir, base+".png")

		job := MarkdownJob{
			SourceFile: path,
			ImageJob: ImageJob{
				OGData: OGData{
					Tag:         og.Tag,
					Title:       og.Title,
					Description: og.Description,
					SiteName:    og.SiteName,
				},
				Template: og.Template,
				Out:      outPath,
				Raw:      og.Raw,
			},
		}

		jobs = append(jobs, job)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", root, err)
	}

	return jobs, nil
}

// parseFrontmatter reads a markdown file and extracts YAML frontmatter
// delimited by "---" lines. Returns nil if no frontmatter is found.
func parseFrontmatter(path string) (*frontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// First line must be "---"
	if !scanner.Scan() {
		return nil, nil
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, nil
	}

	// Read until closing "---"
	var yamlLines []string
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			found = true
			break
		}
		yamlLines = append(yamlLines, line)
	}

	if !found {
		return nil, nil
	}

	yamlContent := strings.Join(yamlLines, "\n")

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
	}

	return &fm, nil
}
