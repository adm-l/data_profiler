package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/go-data-profiler/internal/domain"
	"github.com/example/go-data-profiler/internal/store"
	"github.com/example/go-data-profiler/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Dependencies struct {
	Store  store.Store
	Jobs   *worker.Manager
	Logger *zap.Logger
}

type jobResponse struct {
	ID         string                 `json:"id"`
	Status     string                 `json:"status"`
	Error      string                 `json:"error,omitempty"`
	Request    profileRequestResponse `json:"request"`
	CreatedAt  interface{}            `json:"created_at"`
	StartedAt  interface{}            `json:"started_at,omitempty"`
	FinishedAt interface{}            `json:"finished_at,omitempty"`
}

type profileRequestResponse struct {
	Source       sourceResponse        `json:"source"`
	Schema       string                `json:"schema"`
	Table        string                `json:"table"`
	SampleSize   int                   `json:"sample_size,omitempty"`
	QualityRules *domain.QualityRules  `json:"quality_rules,omitempty"`
}

type sourceResponse struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

func sanitizeJob(j *domain.Job) jobResponse {
	return jobResponse{
		ID: j.ID,
		Status: j.Status,
		Error: j.Error,
		Request: profileRequestResponse{
			Source: sourceResponse{Type: j.Request.Source.Type, DSN: "[REDACTED]"},
			Schema: j.Request.Schema,
			Table: j.Request.Table,
			SampleSize: j.Request.SampleSize,
			QualityRules: j.Request.QualityRules,
		},
		CreatedAt: j.CreatedAt,
		StartedAt: j.StartedAt,
		FinishedAt: j.FinishedAt,
	}
}

func NewRouter(d Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/profile-jobs", func(w http.ResponseWriter, req *http.Request) {
			var in domain.ProfileRequest
			if json.NewDecoder(req.Body).Decode(&in) != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(in.Source.Type) == "" || strings.TrimSpace(in.Source.DSN) == "" || in.Schema == "" || in.Table == "" {
				http.Error(w, "source.type, source.dsn, schema and table are required", http.StatusBadRequest)
				return
			}
			key := strings.TrimSpace(req.Header.Get("Idempotency-Key"))
			j, err := d.Jobs.Create(in, key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			write(w, http.StatusAccepted, sanitizeJob(j))
		})
		r.Get("/profile-jobs/{id}", func(w http.ResponseWriter, req *http.Request) {
			j, err := d.Store.GetJob(req.Context(), chi.URLParam(req, "id"))
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			write(w, http.StatusOK, sanitizeJob(j))
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
			p, err := d.Store.GetLatestProfile(req.Context(), chi.URLParam(req, "id"))
			if err != nil {
				http.Error(w, "result not found", http.StatusNotFound)
				return
			}
			write(w, http.StatusOK, p)
		})
	})
	return r
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
