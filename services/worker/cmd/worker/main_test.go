package main

import (
	"context"
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/services/worker/internal/outbox"
	"github.com/jackc/pgx/v5"
)

type workerPoolStub struct {
	closed bool
}

func (pool *workerPoolStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func (pool *workerPoolStub) Close() {
	pool.closed = true
}

type processorStep struct {
	result outbox.ProcessResult
	err    error
}

type nextProcessorStub struct {
	steps  []processorStep
	cancel context.CancelFunc
	calls  int
}

func (processor *nextProcessorStub) ProcessNext(context.Context) (outbox.ProcessResult, error) {
	if processor.calls >= len(processor.steps) {
		panic("unexpected ProcessNext call")
	}
	step := processor.steps[processor.calls]
	processor.calls++
	if processor.calls == len(processor.steps) && processor.cancel != nil {
		processor.cancel()
	}
	return step.result, step.err
}

func TestRunWorkerOwnsPoolLifecycleWhenProcessingDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &workerPoolStub{}
	openCalls := 0
	openPool := func(_ context.Context, dsn string) (workerPool, error) {
		openCalls++
		if dsn != "postgres://worker" {
			t.Fatalf("pool DSN = %q", dsn)
		}
		cancel()
		return pool, nil
	}
	getenv := func(key string) string {
		switch key {
		case "POSTGRES_DSN":
			return "postgres://worker"
		case workerProcessingEnabledEnvironment:
			return "false"
		default:
			return ""
		}
	}
	if err := runWorker(ctx, getenv, openPool); err != nil {
		t.Fatalf("runWorker() error = %v", err)
	}
	if openCalls != 1 || !pool.closed {
		t.Fatalf("pool lifecycle = opens %d, closed %t", openCalls, pool.closed)
	}
}

func TestRunWorkerFailsClosedBeforeProcessing(t *testing.T) {
	validEnvironment := func(key string) string {
		if key == "POSTGRES_DSN" {
			return "postgres://worker"
		}
		return "false"
	}
	tests := map[string]struct {
		ctx      context.Context
		getenv   func(string) string
		openPool workerPoolOpener
	}{
		"nil context": {getenv: validEnvironment, openPool: func(context.Context, string) (workerPool, error) {
			return &workerPoolStub{}, nil
		}},
		"nil pool opener": {ctx: context.Background(), getenv: validEnvironment},
		"invalid config": {
			ctx: context.Background(), getenv: func(string) string { return "" },
			openPool: func(context.Context, string) (workerPool, error) {
				t.Fatal("invalid config opened a pool")
				return nil, nil
			},
		},
		"pool failure": {
			ctx: context.Background(), getenv: validEnvironment,
			openPool: func(context.Context, string) (workerPool, error) {
				return nil, errors.New("database unavailable")
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := runWorker(test.ctx, test.getenv, test.openPool); !errors.Is(err, ErrWorkerUnavailable) {
				t.Fatalf("runWorker() error = %v, want ErrWorkerUnavailable", err)
			}
		})
	}
}

func TestRunProcessingLoopDrainsThenStopsWithoutDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &nextProcessorStub{
		steps: []processorStep{
			{result: outbox.ProcessCompleted},
			{result: outbox.ProcessRetryScheduled},
			{result: outbox.ProcessDiscarded},
			{result: outbox.ProcessNoEvent},
		},
		cancel: cancel,
	}
	if err := runProcessingLoop(ctx, processor); err != nil {
		t.Fatalf("runProcessingLoop() error = %v", err)
	}
	if processor.calls != len(processor.steps) {
		t.Fatalf("ProcessNext calls = %d, want %d", processor.calls, len(processor.steps))
	}
}

func TestRunProcessingLoopStopsAfterProcessingFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &nextProcessorStub{
		steps:  []processorStep{{err: errors.New("sensitive handler failure")}},
		cancel: cancel,
	}
	if err := runProcessingLoop(ctx, processor); err != nil {
		t.Fatalf("runProcessingLoop() error = %v", err)
	}
	if processor.calls != 1 {
		t.Fatalf("ProcessNext calls = %d, want 1", processor.calls)
	}
}

func TestHandleM2Revocation(t *testing.T) {
	for _, kind := range []outbox.EventKind{outbox.EventDeviceRevoked, outbox.EventAgentRevoked} {
		if err := handleM2Revocation(context.Background(), kind, "opaque-id"); err != nil {
			t.Fatalf("handleM2Revocation(%v) error = %v", kind, err)
		}
	}
	if err := handleM2Revocation(context.Background(), outbox.EventKind(255), "opaque-id"); !errors.Is(err, outbox.ErrPermanentHandler) {
		t.Fatalf("handleM2Revocation(unknown) error = %v", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := handleM2Revocation(canceledCtx, outbox.EventDeviceRevoked, "opaque-id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("handleM2Revocation(canceled) error = %v", err)
	}
}
