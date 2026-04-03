package web

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"fmtDuration": fmtDuration,
	"fmtDate":     fmtDate,
	"fmtDatePtr":  fmtDatePtr,
	"add":         func(a, b int) int { return a + b },
	"sub":         func(a, b int) int { return a - b },
	"mul":         func(a, b int) int { return a * b },
	"hasPages":    func(total, perPage int) bool { return total > perPage },
	"fmtSize":     fmtSize,
	"ftoi":        func(f float64) int { return int(f) },
	"linkify":     linkify,
	"ogDesc":      ogDesc,
	"webPath":     webPath,
	"currentURL":  func() string { return "" },
	"loggedIn":    func() bool { return false },
	"authEnabled": func() bool { return true },
}

func fmtDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func fmtDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func fmtDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

func fmtSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

var urlRe = regexp.MustCompile(`(https?://[^\s<>"'` + "`" + `\x00-\x1f]+)`)
var tagRe = regexp.MustCompile(`<[^>]*>`)

func linkify(text string) template.HTML {
	escaped := html.EscapeString(text)
	result := urlRe.ReplaceAllStringFunc(escaped, func(u string) string {
		return `<a href="` + u + `" target="_blank" rel="noopener">` + u + `</a>`
	})
	return template.HTML(result)
}

func ogDesc(text string) string {
	charLimit := 150
	linked := string(linkify(text))
	plain := html.UnescapeString(tagRe.ReplaceAllString(linked, ""))
	if len(plain) > charLimit {
		return plain[:charLimit-3] + "..."
	}
	return plain
}

func webPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.ToSlash(p)
	return strings.TrimLeft(p, "/")
}

type Templates struct {
	templates map[string]*template.Template
}

func NewTemplates(templateDir string) (*Templates, error) {
	basePath := filepath.Join(templateDir, "base.tmpl")

	pages, err := filepath.Glob(filepath.Join(templateDir, "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("globbing templates: %w", err)
	}

	t := &Templates{templates: make(map[string]*template.Template)}
	for _, pagePath := range pages {
		name := filepath.Base(pagePath)
		if name == "base.tmpl" {
			continue
		}
		tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(basePath, pagePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
		t.templates[name] = tmpl
	}
	return t, nil
}

func (t *Templates) Render(w http.ResponseWriter, name string, data any, loggedIn, authEnabled bool) error {
	return t.render(w, name, data, loggedIn, authEnabled, "")
}

func (t *Templates) render(w http.ResponseWriter, name string, data any, loggedIn, authEnabled bool, currentURL string) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	clone, err := tmpl.Clone()
	if err != nil {
		return fmt.Errorf("cloning template %s: %w", name, err)
	}
	clone.Funcs(template.FuncMap{
		"loggedIn":    func() bool { return loggedIn },
		"authEnabled": func() bool { return authEnabled },
		"currentURL":  func() string { return currentURL },
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return clone.ExecuteTemplate(w, "base", data)
}
