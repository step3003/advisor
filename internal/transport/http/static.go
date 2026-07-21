package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaFileServer раздаёт собранный SPA из webDir. Существующие файлы отдаются
// как есть; для остальных путей (клиентские маршруты) возвращается index.html.
func spaFileServer(webDir string) http.Handler {
	fs := http.FileServer(http.Dir(webDir))
	index := filepath.Join(webDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		path := filepath.Join(webDir, clean)
		// Защита от выхода за пределы webDir.
		if !strings.HasPrefix(path, filepath.Clean(webDir)) {
			http.NotFound(w, r)
			return
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
