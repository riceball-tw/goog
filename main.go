package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// OGData holds the template variables injected into the HTML.
type OGData struct {
	Tag         string `json:"tag"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SiteName    string `json:"site_name"`
}

// ImageJob describes a single OG image to generate.
type ImageJob struct {
	OGData
	Template string `json:"template"` // path to the HTML template (optional, uses default)
	Out      string `json:"out"`      // output image path
	Raw      bool   `json:"raw"`      // treat template as raw HTML
}

func main() {
	// CLI flags
	tmplPath := flag.String("template", "templates/og.html", "path to the HTML template")
	outPath := flag.String("out", "og.png", "output image path")
	title := flag.String("title", "Hello, Open Graph!", "og title text")
	description := flag.String("desc", "A simple OG image generator powered by Go and chromedp.", "og description text")
	tag := flag.String("tag", "Blog Post", "tag / category label")
	siteName := flag.String("site", "example.com", "site name shown in footer")
	rawHTML := flag.Bool("raw", false, "treat the template as raw HTML (skip Go template rendering)")
	configPath := flag.String("config", "", "path to a JSON config file for batch generation")
	workers := flag.Int("workers", 4, "number of concurrent workers for batch generation")
	flag.Parse()

	var jobs []ImageJob

	if *configPath != "" {
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
		// Single-image mode (backwards compatible)
		jobs = []ImageJob{
			{
				OGData: OGData{
					Tag:         *tag,
					Title:       *title,
					Description: *description,
					SiteName:    *siteName,
				},
				Template: *tmplPath,
				Out:      *outPath,
				Raw:      *rawHTML,
			},
		}
	}

	if len(jobs) == 0 {
		log.Fatal("no image jobs to process")
	}

	// Create a shared browser allocator so all goroutines reuse one Chrome process
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
		)...,
	)
	defer allocCancel()

	// Create the parent browser context (launches Chrome once)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Ensure the browser is started before spawning goroutines
	if err := chromedp.Run(browserCtx); err != nil {
		log.Fatalf("failed to start browser: %v", err)
	}

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	start := time.Now()

	for i, job := range jobs {
		wg.Add(1)
		sem <- struct{}{} // acquire slot

		go func(idx int, j ImageJob) {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			if err := processJob(browserCtx, j); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("job %d (%s): %w", idx, j.Out, err))
				mu.Unlock()
				log.Printf("❌ Job %d failed (%s): %v", idx, j.Out, err)
			} else {
				log.Printf("✅ [%d/%d] saved %s", idx+1, len(jobs), j.Out)
			}
		}(i, job)
	}

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\n🏁 Generated %d/%d images in %s\n", len(jobs)-len(errors), len(jobs), elapsed.Round(time.Millisecond))

	if len(errors) > 0 {
		fmt.Println("\n⚠️  Errors:")
		for _, e := range errors {
			fmt.Printf("  • %v\n", e)
		}
		os.Exit(1)
	}
}

// processJob renders a single OG image: template → HTML → screenshot → file.
func processJob(browserCtx context.Context, job ImageJob) error {
	// Read the template file
	tmplBytes, err := os.ReadFile(job.Template)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", job.Template, err)
	}

	var htmlContent string

	if job.Raw {
		htmlContent = string(tmplBytes)
	} else {
		t, err := template.New("og").Parse(string(tmplBytes))
		if err != nil {
			return fmt.Errorf("failed to parse template: %w", err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, job.OGData); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}
		htmlContent = buf.String()
	}

	// Each goroutine gets its own tab (context) within the shared browser
	tabCtx, tabCancel := chromedp.NewContext(browserCtx)
	defer tabCancel()

	tabCtx, timeoutCancel := context.WithTimeout(tabCtx, 30*time.Second)
	defer timeoutCancel()

	png, err := capture(tabCtx, htmlContent)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	if err := os.WriteFile(job.Out, png, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", job.Out, err)
	}

	return nil
}

// capture sets the HTML via page.SetDocumentContent and takes a full-page
// screenshot at 1200×630 (standard OG dimensions).
func capture(ctx context.Context, htmlContent string) ([]byte, error) {
	var buf []byte

	err := chromedp.Run(ctx,
		// Navigate to about:blank so we have a valid frame
		chromedp.Navigate("about:blank"),

		// Set the viewport to OG image dimensions (1200×630)
		chromedp.EmulateViewport(1200, 630),

		// Inject our HTML using CDP Page.SetDocumentContent
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to get frame tree: %w", err)
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlContent).Do(ctx)
		}),

		// Wait for the next paint frame to ensure rendering is complete
		chromedp.Evaluate(`new Promise(r => requestAnimationFrame(r))`, nil),

		// Take a full-page screenshot
		chromedp.FullScreenshot(&buf, 100),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp run failed: %w", err)
	}

	return buf, nil
}
