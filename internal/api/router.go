package api

import (
	"encoding/json"
	"github.com/example/go-data-profiler/internal/domain"
	"github.com/example/go-data-profiler/internal/store"
	"github.com/example/go-data-profiler/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
)

type Dependencies struct {
	Store  store.Store
	Jobs   *worker.Manager
	Logger *zap.Logger
}

func NewRouter(d Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); w.Write([]byte(`{"status":"ok"}`)) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ready"}`))
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/profile-jobs", func(w http.ResponseWriter, req *http.Request) {
			var in domain.ProfileRequest
			if json.NewDecoder(req.Body).Decode(&in) != nil {
				http.Error(w, "invalid JSON", 400)
				return
			}
			if strings.TrimSpace(in.Source.Type) == "" || strings.TrimSpace(in.Source.DSN) == "" || in.Schema == "" || in.Table == "" {
				http.Error(w, "source.type, source.dsn, schema and table are required", 400)
				return
			}
			j, e := d.Jobs.Create(in)
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			write(w, 202, j)
		})
		r.Get("/profile-jobs/{id}", func(w http.ResponseWriter, req *http.Request) {
			j, e := d.Store.GetJob(req.Context(), chi.URLParam(req, "id"))
			if e != nil {
				http.Error(w, "not found", 404)
				return
			}
			write(w, 200, j)
		})
		r.Get("/profiles/history", func(w http.ResponseWriter, req *http.Request) {
			schema := strings.TrimSpace(req.URL.Query().Get("schema"))
			table := strings.TrimSpace(req.URL.Query().Get("table"))
			if schema == "" || table == "" {
				http.Error(w, "schema and table are required", http.StatusBadRequest)
				return
			}
			limit := 20
			if raw := req.URL.Query().Get("limit"); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 1 || n > 100 {
					http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
					return
				}
				limit = n
			}
			profiles, err := d.Store.GetProfileHistory(req.Context(), schema, table, limit)
			if err != nil {
				http.Error(w, "history not found", http.StatusNotFound)
				return
			}
			write(w, http.StatusOK, profiles)
		})
		r.Get("/profiles/drift", func(w http.ResponseWriter, req *http.Request) {
			schema := strings.TrimSpace(req.URL.Query().Get("schema"))
			table := strings.TrimSpace(req.URL.Query().Get("table"))
			if schema == "" || table == "" {
				http.Error(w, "schema and table are required", http.StatusBadRequest)
				return
			}
			profiles, err := d.Store.GetProfileHistory(req.Context(), schema, table, 1)
			if err != nil || len(profiles) == 0 || profiles[0].Drift == nil {
				http.Error(w, "drift not found", http.StatusNotFound)
				return
			}
			write(w, http.StatusOK, profiles[0].Drift)
		})
		r.Get("/profile-jobs/{id}/result", func(w http.ResponseWriter, req *http.Request) {
			p, e := d.Store.GetLatestProfile(req.Context(), chi.URLParam(req, "id"))
			if e != nil {
				http.Error(w, "result not found", 404)
				return
			}
			write(w, 200, p)
		})
	})
	return r
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
