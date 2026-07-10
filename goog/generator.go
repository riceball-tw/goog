package goog

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ImageJob describes a single OG image to generate.
type ImageJob struct {
	Vars     map[string]string `json:"vars" yaml:"vars"`     // template variables (any keys)
	Template string            `json:"template" yaml:"template"` // path to the HTML template (optional, uses default)
	Out      string            `json:"out" yaml:"out"`       // output image path
	Raw      bool              `json:"raw" yaml:"raw"`       // treat template as raw HTML
}

// Generator manages a shared Chrome browser for OG image generation.
type Generator struct {
	workers     int
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	browserCancel context.CancelFunc
}

// New creates a new Generator with a shared Chrome allocator.
// Call Close() when done to release Chrome resources.
func New(workers int) (*Generator, error) {
	if workers < 1 {
		workers = 1
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)...,
	)

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	// Ensure the browser is started
	if err := chromedp.Run(browserCtx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	return &Generator{
		workers:       workers,
		allocCtx:      allocCtx,
		allocCancel:   allocCancel,
		browserCtx:    browserCtx,
		browserCancel: browserCancel,
	}, nil
}

// Generate processes a batch of ImageJobs concurrently using the worker pool.
func (g *Generator) Generate(ctx context.Context, jobs []ImageJob) error {
	if len(jobs) == 0 {
		return fmt.Errorf("no image jobs to process")
	}

	sem := make(chan struct{}, g.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	start := time.Now()

	for i, job := range jobs {
		wg.Add(1)
		sem <- struct{}{} // acquire slot

		go func(idx int, j ImageJob) {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			if err := g.processJob(ctx, j, idx, len(jobs)); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("job %d (%s): %w", idx, j.Out, err))
				mu.Unlock()
				log.Printf("❌ Job %d failed (%s): %v", idx, j.Out, err)
			}
		}(i, job)
	}

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\n🏁 Generated %d/%d images in %s\n", len(jobs)-len(errs), len(jobs), elapsed.Round(time.Millisecond))

	if len(errs) > 0 {
		fmt.Println("\n⚠️  Errors:")
		for _, e := range errs {
			fmt.Printf("  • %v\n", e)
		}
		return fmt.Errorf("%d job(s) failed", len(errs))
	}

	return nil
}

// processJob renders a single OG image: template → HTML → screenshot → file.
func (g *Generator) processJob(ctx context.Context, job ImageJob, idx, total int) error {
	start := time.Now()

	htmlContent, err := renderHTML(job)
	if err != nil {
		return err
	}

	// Each goroutine gets its own tab (context) within the shared browser
	tabCtx, tabCancel := chromedp.NewContext(g.browserCtx)
	defer tabCancel()

	tabCtx, timeoutCancel := context.WithTimeout(tabCtx, 30*time.Second)
	defer timeoutCancel()

	png, err := captureScreenshot(tabCtx, htmlContent)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	if err := os.WriteFile(job.Out, png, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", job.Out, err)
	}

	elapsed := time.Since(start)
	log.Printf("✅ [%d/%d] [%s] saved %s", idx+1, total, elapsed.Round(time.Millisecond), job.Out)
	return nil
}

// Close releases Chrome resources.
func (g *Generator) Close() {
	if g.browserCancel != nil {
		g.browserCancel()
	}
	if g.allocCancel != nil {
		g.allocCancel()
	}
}
