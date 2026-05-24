GO ?= go
NPM ?= npm

CLI_ARGS ?= help
TROVE_DATABASE_URL ?= postgres://trove:trove@localhost:5432/trove?sslmode=disable
TROVE_CLI_INSTALL_PATH ?= /usr/local/bin/trove

.PHONY: help
help:
	@printf 'Trove development targets:\n'
	@printf '  make server              Run the Trove server with seeded in-memory data\n'
	@printf '  make server-db           Run the Trove server with local PostgreSQL and dev auth\n'
	@printf '  make cli CLI_ARGS="help" Run the Trove CLI through go run\n'
	@printf '  make install-cli         Build and install the Trove CLI to /usr/local/bin/trove\n'
	@printf '  make postgres            Start a local PostgreSQL container\n'
	@printf '  make test                Run Go tests\n'
	@printf '  make build               Build all Go packages\n'
	@printf '  make check               Run Go tests/build plus web/docs builds\n'
	@printf '  make install             Install web and docs dependencies\n'
	@printf '  make web-dev             Run the React/Vite app\n'
	@printf '  make web-build           Build the embedded React/Vite app\n'
	@printf '  make web-preview         Preview the built React/Vite app\n'
	@printf '  make docs-dev            Run the VitePress docs site\n'
	@printf '  make docs-build          Build the VitePress docs site\n'
	@printf '  make docs-preview        Preview the built VitePress docs site\n'

.PHONY: server
server:
	$(GO) run ./cmd/trove

.PHONY: server-db
server-db:
	TROVE_DATABASE_URL='$(TROVE_DATABASE_URL)' \
	TROVE_DATABASE_MIGRATE_ON_STARTUP=true \
	TROVE_AUTH_MODE=dev \
	TROVE_AUTH_DEV_MODE_ENABLED=true \
	$(GO) run ./cmd/trove

.PHONY: cli
cli:
	$(GO) run ./cmd/trove $(CLI_ARGS)

.PHONY: install-cli
install-cli:
	$(GO) build -o /tmp/trove ./cmd/trove
	sudo install -m 0755 /tmp/trove $(TROVE_CLI_INSTALL_PATH)

.PHONY: postgres
postgres:
	docker run --rm --name trove-postgres \
		-e POSTGRES_USER=trove \
		-e POSTGRES_PASSWORD=trove \
		-e POSTGRES_DB=trove \
		-p 5432:5432 \
		postgres:16

.PHONY: test
test:
	$(GO) test ./...

.PHONY: build
build:
	$(GO) build ./...

.PHONY: check
check: test build web-build docs-build

.PHONY: install
install: web-install docs-install

.PHONY: web-install
web-install:
	$(NPM) --prefix web ci

.PHONY: web-dev web
web-dev web:
	$(NPM) --prefix web run dev

.PHONY: web-build
web-build:
	$(NPM) --prefix web run build

.PHONY: web-preview
web-preview:
	$(NPM) --prefix web run preview

.PHONY: docs-install
docs-install:
	$(NPM) --prefix docs-site ci

.PHONY: docs-dev docs docsite
docs-dev docs docsite:
	$(NPM) --prefix docs-site run docs:dev

.PHONY: docs-build
docs-build:
	$(NPM) --prefix docs-site run docs:build

.PHONY: docs-preview
docs-preview:
	$(NPM) --prefix docs-site run docs:preview
