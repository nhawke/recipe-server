package recipes

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	_ "embed"
)

var (
	//go:embed recipe.html.tmpl
	recipeTmplText string

	md = goldmark.New(
		goldmark.WithExtensions(extension.Linkify),
	)

	DefaultRecipeTemplate = template.Must(template.New("recipe").Parse(recipeTmplText))
)

// Handler is a HTTP handler for serving a recipes directory.
type Handler struct {
	// Path of the recipe directory
	Path string
	// Template to be used for rendering a recipe page. Each recipe page uses the
	// Page struct for its data.
	RecipeTemplate *template.Template
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	page := req.URL.Path

	if page == "/" {
		h.dirList(w, req)
		return
	}

	eprintf("Request path: %v\n", page)
	if lpage := strings.ToLower(page); !(strings.HasSuffix(lpage, "/") || strings.HasSuffix(lpage, ".md")) {
		page += ".md"
	}

	fpath := path.Join(h.Path, page)
	eprintf("Serving page: %v\n", fpath)
	h.serveFile(w, req, path.Clean(fpath))
}

// dirList lists the contents of a directory, excluding the .md
// suffix and hidden files with a . prefix.
func (h *Handler) dirList(w http.ResponseWriter, r *http.Request) {
	dir, err := os.ReadDir(h.Path)

	if err != nil {
		errString := fmt.Sprintf("Error reading recipe folder %q: %v\n", h.Path, err)
		http.Error(w, errString, 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<pre>\n")

	for _, dirE := range dir {
		name := strings.TrimSuffix(dirE.Name(), ".md")

		// skip hidden files
		if name[0] == '.' {
			continue
		}

		// Properly serialize URL
		url := url.URL{Path: name}
		fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", url.String(), name)
	}

	fmt.Fprintf(w, "</pre>\n")
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, fs.ErrPermission) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var buf bytes.Buffer
	if err := md.Convert(b, &buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	page := Page{
		Title: pageTitle(path),
		Body:  template.HTML(buf.String()),
	}

	if err := h.RecipeTemplate.Execute(w, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Page struct {
	Title string
	Body  template.HTML
}

func pageTitle(p string) string {
	file := path.Base(p)
	name, _, _ := strings.Cut(file, ".")
	return name
}

func eprintf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func eprintln(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}

func exit(format string, args ...any) {
	eprintf(format, args...)
	os.Exit(1)
}
