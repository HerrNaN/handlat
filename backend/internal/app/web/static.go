package web

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed all:dist
var dist embed.FS

func ServeStatic(appVersion string) (http.HandlerFunc, error) {
	staticFS, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("create sub fs: %w", err)
	}

	index, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read index html: %w", err)
	}

	const versionMeta = `<meta name="handlat-version" content="`

	indexHTML := strings.Replace(
		string(index),
		versionMeta+`"`,
		versionMeta+html.EscapeString(appVersion)+`"`,
		1,
	)
	fileServer := http.FileServer(http.FS(staticFS))

	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(staticFS, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r.URL.Path = "/"
		}

		w.Header().Set("Cache-Control", cacheControl(r.URL.Path))

		if r.URL.Path == "/" {
			http.ServeContent(w, r, "index.html", time.Time{}, strings.NewReader(indexHTML))
			return
		}

		fileServer.ServeHTTP(w, r)
	}, nil
}

func cacheControl(path string) string {
	switch {
	case strings.HasPrefix(path, "/assets/"):
		return "public, max-age=31536000, immutable"
	case path == "/index.html" || path == "/":
		return "no-cache"
	default:
		return "no-cache"
	}
}
