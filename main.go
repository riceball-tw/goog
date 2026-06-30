package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// OGData holds the template variables injected into the HTML.
type OGData struct {
	Tag         string
	Title       string
	Description string
	SiteName    string
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
	flag.Parse()

	// Read the template file
	tmplBytes, err := os.ReadFile(*tmplPath)
	if err != nil {
		log.Fatalf("failed to read template %s: %v", *tmplPath, err)
	}

	var htmlContent string

	if *rawHTML {
		// Use the file contents as-is (e.g. a plain index.html)
		htmlContent = string(tmplBytes)
	} else {
		// Execute Go template
		t, err := template.New("og").Parse(string(tmplBytes))
		if err != nil {
			log.Fatalf("failed to parse template: %v", err)
		}
		data := OGData{
			Tag:         *tag,
			Title:       *title,
			Description: *description,
			SiteName:    *siteName,
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			log.Fatalf("failed to execute template: %v", err)
		}
		htmlContent = buf.String()
	}

	// Capture the screenshot
	png, err := capture(htmlContent)
	if err != nil {
		log.Fatalf("capture failed: %v", err)
	}

	if err := os.WriteFile(*outPath, png, 0644); err != nil {
		log.Fatalf("failed to write %s: %v", *outPath, err)
	}

	fmt.Printf("✅ OG image saved to %s (%d bytes)\n", *outPath, len(png))
}

// capture launches a headless Chrome, sets the HTML via page.SetDocumentContent,
// and takes a full-page screenshot at 1200×630 (standard OG dimensions).
func capture(htmlContent string) ([]byte, error) {
	// Create a headless browser context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a timeout so we don't hang forever
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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
