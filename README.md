# goog — Go Open Graph

> Generate OG images from customizable Go templates
> ```bash
> goog --var title="Hello, World!" --var description="A quick intro" --var tag="Blog" --var site="example.com" --out card.png
> ```

## Why

There's plenty mature og generator out there like: [satori](https://github.com/vercel/satori), but it's not suitable for my need:
- **speed** - JavaScript is slow and single threaded, it always took ~800ms to render single image, which is unbearable for thousands of images  
- **customization** - Without JSX, just build from pure html template

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
| `--var` | — | Template variable (`key=value`, repeatable) |
| `--raw` | `false` | Treat template file as raw HTML (skip Go template engine) |

### Built-in Template

The default template (`templates/og.html`) renders a dark gradient card with these slots:

| Template Variable | `--var` key | Description |
|---|---|---|
| `{{.tag}}` | `tag` | Uppercase tag badge (purple) |
| `{{.title}}` | `title` | Main headline (56px bold) |
| `{{.description}}` | `description` | Subtitle text (24px, muted) |
| `{{.site}}` | `site` | Footer site name |

### Custom Templates & Arbitrary Variables

Use `--template` to point to your own HTML file. The template uses Go's `text/template` syntax and can reference **any** variable you pass via `--var`.

```bash
# Template with any fields you want
goog --template my-card.html \
  --var "title=My Post" \
  --var "author=John Doe" \
  --var "date=2026-07-05" \
  --var "reading_time=8 min"
```

Your template `my-card.html` can then use `{{.title}}`, `{{.author}}`, `{{.date}}`, `{{.reading_time}}`:

```html
<div class="byline">{{.author}} · {{.date}} · {{.reading_time}}</div>
<div class="title">{{.title}}</div>
```

If you run `goog` without any `--var` flags, sensible defaults are used:
`title=Hello, Open Graph!`, `tag=Blog Post`, `description=...`, `site=example.com`.

Use `--raw` to skip Go template rendering entirely and serve the file as-is:

```bash
goog --template static.html --raw
```

## GitHub Action

Generated images are staged in an artifact directory, so the workflow run keeps the images for review without pushing them back into the repository.

```yaml
name: OG Images
on:
  workflow_dispatch:

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - id: generate
        uses: riceball-tw/goog@v1
        with:
          config: images.json
      - uses: actions/upload-artifact@v4
        with:
          name: og-images
          path: ${{ steps.generate.outputs.artifact_path }}
          if-no-files-found: error
```

Set `artifact_path` to change the staging directory.

## Example

```bash
goog \
  --var "tag=Tutorial" \
  --var "title=How to Generate OG Images in Go" \
  --var "description=Learn how to automate social card generation with chromedp" \
  --var "site=example.com" \
  --out tutorial-og.png
```

Output: `tutorial-og.png` (1200×630px PNG).

## Why "goog"?

Short for "Go OG" — generates Open Graph images, written in Go.
