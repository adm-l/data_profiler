package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/example/go-data-profiler/internal/adapters"
	"github.com/example/go-data-profiler/internal/domain"
	"github.com/example/go-data-profiler/internal/drift"
	"github.com/example/go-data-profiler/internal/pii"
	"github.com/example/go-data-profiler/internal/quality"
	"github.com/example/go-data-profiler/internal/store"
)

type Manager struct {
	store   store.Store
	jobs    chan string
	workers int
	logger  *zap.Logger
	stop    context.CancelFunc
	wg      sync.WaitGroup
}

func NewManager(s store.Store, n int, l *zap.Logger) *Manager {
	if n < 1 {
		n = 1
	}
	return &Manager{store: s, jobs: make(chan string, 1000), workers: n, logger: l}
}

func (m *Manager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.stop = cancel
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.loop(ctx)
	}
}
func (m *Manager) Stop() {
	if m.stop != nil {
		m.stop()
	}
	m.wg.Wait()
}
func (m *Manager) Enqueue(id string) bool {
	select {
	case m.jobs <- id:
		return true
	default:
		m.logger.Warn("job queue full", zap.String("job_id", id))
		return false
	}
}

func (m *Manager) Create(req domain.ProfileRequest) (*domain.Job, error) {
	j := &domain.Job{ID: uuid.NewString(), Status: "queued", Request: req, CreatedAt: time.Now().UTC()}
	ctx := context.Background()
	if err := m.store.CreateJob(ctx, j); err != nil {
		return nil, err
	}
	if !m.Enqueue(j.ID) {
		now := time.Now().UTC()
		j.Status = "failed"
		j.Error = "job queue is full; retry later"
		j.FinishedAt = &now
		if err := m.store.UpdateJob(ctx, j); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("job queue is full; retry later")
	}
	return j, nil
}
func (m *Manager) loop(ctx context.Context) {
	defer m.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-m.jobs:
			m.run(ctx, id)
		}
	}
}
func (m *Manager) run(parent context.Context, id string) {
	j, err := m.store.GetJob(parent, id)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	j.Status = "running"
	j.StartedAt = &now
	_ = m.store.UpdateJob(parent, j)
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	a, err := adapters.Open(ctx, j.Request.Source.Type, j.Request.Source.DSN)
	if err == nil {
		defer a.Close()
		var p *domain.TableProfile
		p, err = a.ProfileTable(ctx, j.Request.Schema, j.Request.Table, j.Request.SampleSize)
		if err == nil {
			for i := range p.Columns {
				detection := pii.DetectProfile(p.Columns[i].Name, p.Columns[i].DataType, p.Columns[i].TopValues)
				p.Columns[i].PII = detection.Label
				p.Columns[i].PIIConfidence = detection.Confidence
			}
			p.Quality = quality.Evaluate(p, j.Request.QualityRules)
			p.CreatedAt = time.Now().UTC()

			// Compare this snapshot with the most recent completed profile
			// for the same schema/table before persisting the new snapshot.
			if previous, previousErr := m.store.GetPreviousProfile(ctx, p.Schema, p.Table); previousErr == nil {
				report := drift.Compare(*previous, *p)
				p.Drift = &report
			}

			err = m.store.SaveProfile(ctx, j.ID, p)
		}
	}
	if err != nil {
		j.Status = "failed"
		j.Error = fmt.Sprintf("%v", err)
	} else {
		j.Status = "completed"
	}
	end := time.Now().UTC()
	j.FinishedAt = &end
	_ = m.store.UpdateJob(context.Background(), j)
}
