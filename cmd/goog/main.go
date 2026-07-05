package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/riceball-tw/goog/goog"
)

// varMap implements flag.Value for repeatable --var key=val flags.
type varMap map[string]string

func (v varMap) String() string { return "" }
func (v varMap) Set(s string) error {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --var format: %q (expected key=value)", s)
	}
	v[parts[0]] = parts[1]
	return nil
}

func main() {
	// CLI flags
	tmplPath := flag.String("template", "templates/og.html", "path to the HTML template")
	outPath := flag.String("out", "og.png", "output image path")
	rawHTML := flag.Bool("raw", false, "treat the template as raw HTML (skip Go template rendering)")
	configPath := flag.String("config", "", "path to a JSON config file for batch generation")
	workers := flag.Int("workers", 4, "number of concurrent workers for batch generation")
	scanMarkdown := flag.String("scan-markdown", "", "directory to scan for markdown files with ogImage frontmatter")
	ignorePatterns := flag.String("ignore-patterns", "", "comma-separated glob patterns to ignore (markdown mode)")
	vars := make(varMap)
	flag.Var(&vars, "var", "template variable (key=value, repeatable)")
	flag.Parse()

	var jobs []goog.ImageJob

	if *scanMarkdown != "" {
		// Markdown scanning mode
		var patterns []string
		if *ignorePatterns != "" {
			patterns = strings.Split(*ignorePatterns, ",")
		}

		mdJobs, err := goog.ScanMarkdown(*scanMarkdown, patterns)
		if err != nil {
			log.Fatalf("failed to scan markdown: %v", err)
		}

		if len(mdJobs) == 0 {
			log.Fatal("no markdown files with ogImage frontmatter found")
		}

		// Fill in default template for jobs that don't specify one
		for _, mj := range mdJobs {
			if mj.Template == "" {
				mj.Template = *tmplPath
			}
			jobs = append(jobs, mj.ImageJob)
		}

		log.Printf("📝 Found %d markdown files with ogImage frontmatter", len(jobs))
	} else if *configPath != "" {
		// Batch mode: read jobs from JSON config
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("failed to read config %s: %v", *configPath, err)
		}
		if err := json.Unmarshal(data, &jobs); err != nil {
			log.Fatalf("failed to parse config JSON: %v", err)
		}
		// Fill in default template path for jobs that don't specify one
		for i := range jobs {
			if jobs[i].Template == "" {
				jobs[i].Template = *tmplPath
			}
		}
	} else {
		// Single-image mode
		if len(vars) == 0 {
			// Sensible defaults so --out og.png just works
			vars["title"] = "Hello, Open Graph!"
			vars["tag"] = "Blog Post"
			vars["description"] = "A simple OG image generator powered by Go and chromedp."
			vars["site"] = "example.com"
		}
		jobs = []goog.ImageJob{
			{
				Vars:     vars,
				Template: *tmplPath,
				Out:      *outPath,
				Raw:      *rawHTML,
			},
		}
	}

	if len(jobs) == 0 {
		log.Fatal("no image jobs to process")
	}

	// Create the generator
	gen, err := goog.New(*workers)
	if err != nil {
		log.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	// Generate all images
	if err := gen.Generate(context.Background(), jobs); err != nil {
		fmt.Fprintf(os.Stderr, "generation failed: %v\n", err)
		os.Exit(1)
	}
}
