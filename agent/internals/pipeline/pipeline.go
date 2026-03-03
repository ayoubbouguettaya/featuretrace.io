package pipeline

import (
	"context"
	"log"
	"sync"

	"featuretrace.io/agent/internals/buffer"
	"featuretrace.io/agent/internals/config"
	"featuretrace.io/agent/internals/enrich"
	"featuretrace.io/agent/internals/input"
	"featuretrace.io/agent/internals/model"
	"featuretrace.io/agent/internals/output"
	"featuretrace.io/agent/internals/parser"
)

// Channel buffer sizes — large enough to absorb bursts without blocking
// producers for too long, small enough to bound memory.
const (
	rawChanSize     = 4096
	recordChanSize  = 2048
	enrichChanSize  = 2048
	batchChanSize   = 64
)

// Pipeline wires together every stage of the FeatureTrace Agent:
//
//	Input → Parser → Enricher → Batcher → Exporter
//
// Each stage runs in its own goroutine, connected by typed Go channels.
// This mirrors the architecture of production agents like Fluent Bit.
type Pipeline struct {
	cfg      *config.Config
	input    input.Input
	enricher *enrich.Enricher
	batcher  *buffer.Batcher
	exporter output.Exporter

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New constructs a Pipeline from the loaded Config.
func New(cfg *config.Config) *Pipeline {
	// --- Resolve input ---
	var inp input.Input
	for _, ic := range cfg.Inputs {
		switch ic.Type {
		case "docker":
			inp = &input.DockerInput{LogRoot: ic.Options["log_root"]}
		default:
			log.Printf("[pipeline] unknown input type %q, skipping", ic.Type)
		}
	}
	if inp == nil {
		// Fallback to Docker input with defaults
		inp = &input.DockerInput{}
	}

	// --- Enricher ---
	enricher := enrich.New(nil)

	// --- Batcher ---
	batcher := buffer.New(cfg.Agent.BatchSize, cfg.Agent.FlushInterval)

	// --- Exporter ---
	exp := output.NewHTTPExporter(
		cfg.Output.Endpoint,
		cfg.Output.Timeout,
		cfg.Output.MaxRetries,
		cfg.Output.Compression,
	)

	return &Pipeline{
		cfg:      cfg,
		input:    inp,
		enricher: enricher,
		batcher:  batcher,
		exporter: exp,
	}
}

// Start launches all pipeline stages as concurrent goroutines.
//
//	┌─────────┐    rawChan    ┌────────┐   recordChan   ┌──────────┐
//	│  INPUT   │─────────────→│ PARSER │───────────────→│ ENRICHER │
//	└─────────┘               └────────┘                └──────┬───┘
//	                                                          │ enrichChan
//	                                                          ▼
//	┌──────────┐  batchChan  ┌─────────┐               ┌──────────┐
//	│ EXPORTER │←────────────│ BATCHER │←──────────────│          │
//	└──────────┘             └─────────┘               └──────────┘
func (p *Pipeline) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	rawChan    := make(chan []byte, rawChanSize)
	recordChan := make(chan model.Record, recordChanSize)
	enrichChan := make(chan model.Record, enrichChanSize)
	batchChan  := make(chan []model.Record, batchChanSize)

	// Stage 1 — Input: tails Docker logs, produces raw bytes
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(rawChan)
		log.Println("[pipeline] input stage started")
		if err := p.input.Start(ctx, rawChan); err != nil && ctx.Err() == nil {
			log.Printf("[pipeline] input error: %v", err)
		}
		log.Println("[pipeline] input stage stopped")
	}()

	// Stage 2 — Parser: raw bytes → structured Record
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(recordChan)
		log.Println("[pipeline] parser stage started")
		for raw := range rawChan {
			rec := parser.Parse(raw)
			select {
			case recordChan <- rec:
			case <-ctx.Done():
				return
			}
		}
		log.Println("[pipeline] parser stage stopped")
	}()

	// Stage 3 — Enricher: attach metadata + trace correlation
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(enrichChan)
		log.Println("[pipeline] enrichment stage started")
		for rec := range recordChan {
			p.enricher.Enrich(&rec)
			select {
			case enrichChan <- rec:
			case <-ctx.Done():
				return
			}
		}
		log.Println("[pipeline] enrichment stage stopped")
	}()

	// Stage 4 — Batcher: collects records, flushes on size / time
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(batchChan)
		log.Println("[pipeline] batcher stage started")
		p.batcher.Run(ctx, enrichChan, batchChan)
		log.Println("[pipeline] batcher stage stopped")
	}()

	// Stage 5 — Exporter: sends batches to FeatureTrace backend
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		log.Println("[pipeline] exporter stage started")
		for batch := range batchChan {
			if err := p.exporter.Send(ctx, batch); err != nil {
				log.Printf("[pipeline] export error (batch dropped): %v", err)
			}
		}
		log.Println("[pipeline] exporter stage stopped")
	}()

	log.Println("[pipeline] all stages running")
}

// Stop initiates a graceful shutdown: cancels the context, then waits for
// every stage to drain and exit.
func (p *Pipeline) Stop() {
	log.Println("[pipeline] shutting down…")
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	log.Println("[pipeline] shutdown complete")
}

