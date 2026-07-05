package goog

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// renderHTML reads the template file and renders it with Vars via html/template.
// If job.Raw is true, the template is returned as-is without Go template processing.
func renderHTML(job ImageJob) (string, error) {
	tmplBytes, err := os.ReadFile(job.Template)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", job.Template, err)
	}

	if job.Raw {
		return string(tmplBytes), nil
	}

	t, err := template.New("og").Parse(string(tmplBytes))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, job.Vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// captureScreenshot sets HTML via CDP and takes a screenshot at 1200×630 (standard OG dimensions).
func captureScreenshot(ctx context.Context, htmlContent string) ([]byte, error) {
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
