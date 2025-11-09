package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/berkan-cetinkaya/captcha"
	"github.com/gorilla/mux"
)

type PageData struct {
	Title      string
	SiteKey    string
	Action     string
	Theme      string
	Appearance string
}

func init() {
	LoadEnv()
}

func main() {
	port := "8080"

	r := mux.NewRouter()

	r.HandleFunc("/", renderStatic("index.html")).Methods("GET")

	// API endpoints (CAPTCHA-protected)
	r.Handle("/api/login",
		captcha.Middleware("login")(http.HandlerFunc(loginHandler)),
	).Methods("POST")

	r.Handle("/api/search",
		captcha.Middleware("search")(http.HandlerFunc(searchHandler)),
	).Methods("POST")

	// Demo pages (rendered templates)
	r.HandleFunc("/login", render("login.html", "Login", "login")).Methods("GET")
	r.HandleFunc("/search", render("search.html", "Search", "search")).Methods("GET")

	fmt.Printf("🧩 Demo server running at http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, r)
}

func render(filename, title, action string) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles(resolveTemplatePath(filename)))
	return func(w http.ResponseWriter, r *http.Request) {
		meta, err := captcha.Metadata(action)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load captcha metadata: %v", err), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, PageData{
			Title:      title,
			SiteKey:    meta.SiteKey,
			Action:     action,
			Theme:      meta.Theme,
			Appearance: meta.Appearance,
		})
	}
}

func renderStatic(filename string) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles(resolveTemplatePath(filename)))
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	}
}

func resolveTemplatePath(filename string) string {
	candidates := []string{
		filepath.Join("web", "templates", filename),
		filepath.Join("examples", "demo", "web", "templates", filename),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("web", "templates", filename)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"status":  "login_success",
		"message": "✅ Login success (captcha passed)",
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"status":  "search_success",
		"message": "🔍 Search success (captcha passed)",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
