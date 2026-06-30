# goog — Open Graph Image Generator

Generate Open Graph social card images from Go templates using headless Chrome.

```
goog --title "Hello, World!" --desc "A quick intro to goog" --tag "Blog" --site "example.com" --out card.png
```

## How It Works

1. Renders an HTML template (Go `text/template` or raw HTML) with your text
2. Launches a headless Chrome instance via `chromedp`
3. Takes a 1200×630px full-page screenshot (standard OG image size)
4. Writes the PNG to disk

## Requirements

- Go 1.26+
- Chrome / Chromium installed (used by chromedp under the hood)

## Install

```bash
go install github.com/riceball-tw/goog@latest
```

Or clone and build:

```bash
git clone <repo> && cd goog
go build -o goog .
```

## Usage

```bash
goog [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--template` | `templates/og.html` | Path to the HTML template file |
| `--out` | `og.png` | Output image path |
| `--title` | `"Hello, Open Graph!"` | OG title text |
| `--desc` | `"A simple OG image generator..."` | OG description text |
| `--tag` | `"Blog Post"` | Tag / category label |
| `--site` | `"example.com"` | Site name shown in footer |
| `--raw` | `false` | Treat template file as raw HTML (skip Go template engine) |

### Built-in Template

The default template (`templates/og.html`) renders a dark gradient card with these slots:

| Template Variable | Flag | Description |
|---|---|---|
| `{{.Tag}}` | `--tag` | Uppercase tag badge (purple) |
| `{{.Title}}` | `--title` | Main headline (56px bold) |
| `{{.Description}}` | `--desc` | Subtitle text (24px, muted) |
| `{{.SiteName}}` | `--site` | Footer site name |

### Custom Templates

Use `--template` to point to your own HTML file. The template can use Go's `text/template` syntax with the same `.Tag`, `.Title`, `.Description`, `.SiteName` variables, or use `--raw` to skip Go template rendering entirely and serve the file as-is.

```bash
# With your own Go template
goog --template my-card.html --title "Custom Card"

# With raw HTML (no template processing)
goog --template my-card.html --raw --title ignored
```

When `--raw` is used, no template variables are injected — the file contents are passed directly to Chrome.

## Example

```bash
goog \
  --tag "Tutorial" \
  --title "How to Generate OG Images in Go" \
  --desc "Learn how to automate social card generation with chromedp" \
  --site "example.com" \
  --out tutorial-og.png
```

Output: `tutorial-og.png` (1200×630px PNG).

## Why "goog"?

Short for "Go OG" — generates Open Graph images, written in Go.
