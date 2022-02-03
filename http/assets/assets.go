package assets

import (
	"embed"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Assets contains the web front-end static assets.
//
// Some valid accessors:
//  chesstempo/dist [dir]
//  chesstempo/dist/assets [dir]
//  chesstempo/dist/index.html [file]
//  chesstempo/dist/favicon.ico [file]
//
//go:embed chesstempo/dist/*
var FS embed.FS

func SPAHandler() http.HandlerFunc {
	const (
		root   = "/"
		prefix = "chesstempo/dist"
		index  = "/index.html"
	)
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the absolute path to prevent directory traversal.
		path, err := filepath.Abs(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	fallback:
		// Serve index file when the path is empty.
		if path == root {
			path = index
		}

		// Prepend dist prefix.
		path = filepath.Join(prefix, path)

		// Attempt to open file from the embedded filesystem.
		file, err := FS.Open(path)
		if err != nil {
			// When not found, serve the index so the SPA takes control.
			if os.IsNotExist(err) {
				path = root
				goto fallback
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		frs, ok := file.(io.ReadSeeker)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		http.ServeContent(w, r, path, time.Now(), frs)
	}
}
