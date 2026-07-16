SHELL := /bin/bash

GO_MODULES := \
	services/control-plane \
	services/worker \
	services/relay \
	services/agent \
	packages/go/authn \
	packages/go/cryptox \
	packages/go/ids \
	packages/go/observability \
	packages/go/postgresx \
	packages/go/redisx \
	packages/go/testx \
	protocol/gen/go

.PHONY: help bootstrap bootstrap-mobile proto fmt fmt-go lint lint-go test test-go \
	test-flutter test-web test-infrastructure build build-go build-web up down migrate \
	agent-skills-check

help:
	@echo "bootstrap          Verify required local tools"
	@echo "bootstrap-mobile   Generate local Flutter iOS/Android project files"
	@echo "proto              Lint and generate Go/Dart protocol code"
	@echo "fmt                Format Go, Dart, JS, YAML, and Markdown"
	@echo "lint               Run all configured linters"
	@echo "test               Run all configured test suites"
	@echo "test-go            Run every Go module test suite"
	@echo "test-infrastructure Smoke-test Compose readiness and migrations"
	@echo "build              Build Go binaries, web bundles, and container images"
	@echo "up                 Start PostgreSQL, Redis, control-plane, worker, and relay"
	@echo "down               Stop local services"
	@echo "migrate            Apply control-plane migrations"
	@echo "agent-skills-check Verify Codex/Claude skill mirrors"

bootstrap:
	./scripts/bootstrap.sh

bootstrap-mobile:
	./scripts/bootstrap-mobile.sh

proto:
	@command -v buf >/dev/null || (echo "buf is required"; exit 1)
	buf lint
	rm -rf protocol/gen/go/coderoam
	rm -rf protocol/gen/dart/lib/coderoam
	buf generate
	cd protocol/gen/go && go mod tidy
	cd protocol/gen/dart && dart pub get
fmt: fmt-go
	@if command -v dart >/dev/null; then cd apps/mobile && dart format .; else echo "skip dart format: dart missing"; fi
	@if [ -d node_modules ]; then npm run format:web; else echo "skip web format: run npm install"; fi

fmt-go:
	@for module in $(GO_MODULES); do \
		echo "==> gofmt $$module"; \
		find "$$module" -name '*.go' -type f -print0 | xargs -0 -r gofmt -w; \
	done

lint: lint-go agent-skills-check
	@if command -v buf >/dev/null; then buf lint; else echo "skip buf lint: buf missing"; fi
	@if command -v flutter >/dev/null && [ -d apps/mobile/android ]; then cd apps/mobile && flutter analyze; else echo "skip flutter analyze: run make bootstrap-mobile"; fi
	@if [ -d node_modules ]; then npm run lint:web; else echo "skip web lint: run npm install"; fi

lint-go:
	@for module in $(GO_MODULES); do \
		echo "==> go vet $$module"; \
		(cd "$$module" && go vet ./...); \
	done

test: test-go test-flutter test-web

test-go:
	@for module in $(GO_MODULES); do \
		if find "$$module" -name '*.go' -type f -print -quit | grep -q .; then \
			echo "==> go test $$module"; \
			(cd "$$module" && go test ./...); \
		else \
			echo "==> skip $$module: no generated Go packages yet"; \
		fi; \
	done

test-flutter:
	@if command -v flutter >/dev/null && [ -d apps/mobile/android ]; then cd apps/mobile && flutter test; else echo "skip flutter tests: run make bootstrap-mobile"; fi

test-web:
	@if [ -d node_modules ]; then npm run test:web; else echo "skip web tests: run npm install"; fi

test-infrastructure:
	./scripts/test-smoke-infrastructure.sh
	./scripts/smoke-infrastructure.sh

build: build-go build-web
	docker build -f deployments/docker/control-plane.Dockerfile .
	docker build -f deployments/docker/worker.Dockerfile .
	docker build -f deployments/docker/relay.Dockerfile .
	docker build -f deployments/docker/agent.Dockerfile .

build-go:
	@mkdir -p dist
	@for target in \
		"services/control-plane ./cmd/api dist/coderoam-api" \
		"services/worker ./cmd/worker dist/coderoam-worker" \
		"services/relay ./cmd/relay dist/coderoam-relay" \
		"services/agent ./cmd/coderoam-agent dist/coderoam-agent"; do \
		set -- $$target; \
		echo "==> build $$1"; \
		(cd "$$1" && go build -o "$(CURDIR)/$$3" "$$2"); \
	done

build-web:
	@if [ -d node_modules ]; then npm run build:web; else echo "skip web build: run npm install"; fi

up:
	docker compose -f deployments/compose/docker-compose.yml up --build

down:
	docker compose -f deployments/compose/docker-compose.yml down

migrate:
	./scripts/migrate.sh

agent-skills-check:
	python3 scripts/sync-agent-skills.py --check
