package webassets

import (
	"io/fs"
	"net/http"
)

func WorkbenchHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := StaticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "页面资源不可用", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
}

func StaticHandler() http.Handler {
	sub, err := fs.Sub(StaticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))
}
