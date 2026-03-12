package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"go.abhg.dev/goldmark/mermaid"
)

var md = goldmark.New(goldmark.WithExtensions(
	extension.Table,
	extension.Strikethrough,
	extension.Linkify,
	&mermaid.Extender{},
))

type pageData struct {
	Title string
	Body  template.HTML
}

const pageTpl = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
  body { max-width: 800px; margin: 40px auto; padding: 0 20px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; color: #24292e; line-height: 1.6; }
  a { color: #0366d6; text-decoration: none; }
  a:hover { text-decoration: underline; }
  table { border-collapse: collapse; margin: 16px 0; }
  th, td { border: 1px solid #dfe2e5; padding: 6px 13px; }
  th { background: #f6f8fa; }
  pre { background: #f6f8fa; padding: 16px; overflow-x: auto; border-radius: 6px; }
  code { background: #f6f8fa; padding: 2px 6px; border-radius: 3px; font-size: 85%; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 4px solid #dfe2e5; margin: 0; padding: 0 16px; color: #6a737d; }
  img { max-width: 100%; }
  ul.listing { list-style: none; padding: 0; }
  ul.listing li { padding: 4px 0; }
  .breadcrumb { margin-bottom: 16px; color: #6a737d; }
  .breadcrumb a { color: #0366d6; }
  del { color: #6a737d; }
</style>
</head><body>
{{.Body}}
</body></html>`

var page = template.Must(template.New("page").Parse(pageTpl))

func main() {
	port := flag.Int("p", 8080, "port")
	flag.Parse()

	root := cmp.Or(flag.Arg(0), ".")

	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	root, _ = filepath.Abs(root)

	mux := http.NewServeMux()
	if !info.IsDir() {
		mux.HandleFunc("GET /{$}", singleFileHandler(root))
	} else {
		rootDir, err := os.OpenRoot(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: open root: %v\n", err)
			os.Exit(1)
		}
		defer rootDir.Close()
		mux.HandleFunc("GET /{path...}", dirHandler(rootDir, filepath.Base(root)))
	}

	addr := fmt.Sprintf(":%d", *port)
	fmt.Fprintf(os.Stderr, "Serving %s on http://localhost%s\n", root, addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func singleFileHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		src, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, "read file failed", http.StatusInternalServerError)
			return
		}
		renderMarkdown(w, filepath.Base(filePath), src)
	}
}

func dirHandler(rootDir *os.Root, rootName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlPath := path.Clean("/" + r.PathValue("path"))
		relPath := strings.TrimPrefix(urlPath, "/")

		statPath := cmp.Or(relPath, ".")
		info, err := rootDir.Stat(statPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if info.IsDir() {
			renderDirListing(w, rootDir, rootName, urlPath, statPath)
			return
		}

		if strings.HasSuffix(relPath, ".md") {
			renderMarkdownFromRoot(w, rootDir, relPath)
			return
		}

		http.NotFound(w, r)
	}
}

func renderMarkdownFromRoot(w http.ResponseWriter, rootDir *os.Root, relPath string) {
	f, err := rootDir.Open(relPath)
	if err != nil {
		http.Error(w, "read file failed", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	src, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "read file failed", http.StatusInternalServerError)
		return
	}

	renderMarkdown(w, path.Base(relPath), src)
}

func renderMarkdown(w http.ResponseWriter, title string, src []byte) {
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		http.Error(w, "render markdown failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, pageData{
		Title: title,
		Body:  template.HTML(buf.String()),
	})
}

func renderDirListing(w http.ResponseWriter, rootDir *os.Root, rootName, urlPath, openPath string) {
	f, err := rootDir.Open(openPath)
	if err != nil {
		http.Error(w, "read directory failed", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	entries, err := f.ReadDir(-1)
	if err != nil {
		http.Error(w, "read directory failed", http.StatusInternalServerError)
		return
	}

	var buf strings.Builder

	if urlPath != "/" {
		buf.WriteString(`<div class="breadcrumb">`)
		buf.WriteString(`<a href="/">root</a>`)
		parts := strings.Split(strings.Trim(urlPath, "/"), "/")
		for i, p := range parts {
			link := "/" + strings.Join(parts[:i+1], "/")
			buf.WriteString(` / <a href="` + link + `">` + template.HTMLEscapeString(p) + `</a>`)
		}
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`<ul class="listing">`)

	if urlPath != "/" {
		parent := path.Dir(urlPath)
		buf.WriteString(`<li>📁 <a href="` + parent + `">..</a></li>`)
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		href := path.Join(urlPath, name)

		if e.IsDir() {
			buf.WriteString(`<li>📁 <a href="` + href + `">` + template.HTMLEscapeString(name) + `/</a></li>`)
		} else if strings.HasSuffix(name, ".md") {
			buf.WriteString(`<li>📄 <a href="` + href + `">` + template.HTMLEscapeString(name) + `</a></li>`)
		}
	}

	buf.WriteString(`</ul>`)

	title := path.Base(urlPath)
	if urlPath == "/" {
		title = rootName
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, pageData{
		Title: title,
		Body:  template.HTML(buf.String()),
	})
}
