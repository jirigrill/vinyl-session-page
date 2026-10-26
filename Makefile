# Local development and droplet operations. Run `make` to list targets.

# Droplet SSH target; deploy/deploy.sh defaults to the same value.
HOST ?= root@142.93.99.205
LOCAL_DATA := data/local

# Load .env in the recipe shell (not via make include, so quoted values work).
LOAD_ENV := set -a; [ -f .env ] && . ./.env; set +a

.DEFAULT_GOAL := help
.PHONY: help pull-db run test vet fmt ingest-remote deploy

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

pull-db: ## Copy the production database into data/local/
	@mkdir -p $(LOCAL_DATA)
	@rm -f $(LOCAL_DATA)/vinyl.db $(LOCAL_DATA)/vinyl.db-shm $(LOCAL_DATA)/vinyl.db-wal
	scp -q '$(HOST):/var/lib/vinyl/vinyl.db*' $(LOCAL_DATA)/

run: ## Run the server locally against the snapshot
	@test -f $(LOCAL_DATA)/vinyl.db || { echo "no snapshot; run: make pull-db"; exit 1; }
	@$(LOAD_ENV); \
	export VINYL_USER="$${VINYL_USER:-dev}" VINYL_PASSWORD="$${VINYL_PASSWORD:-dev}" VINYL_DATA_DIR="$(LOCAL_DATA)"; \
	echo "http://$${VINYL_ADDR:-127.0.0.1:8080} (user $$VINYL_USER)"; \
	go run ./cmd/vinyl

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go sources
	gofmt -l -w .

ingest-remote: ## Run ingest on the droplet (needs YT_DATA_API_KEY in .env)
	@$(LOAD_ENV); test -n "$$YT_DATA_API_KEY" || { echo "YT_DATA_API_KEY not set"; exit 1; }; \
	scp -q data/seed-channels.txt '$(HOST):/tmp/seed-channels.txt' && \
	printf '%s\n' "$$YT_DATA_API_KEY" | ssh '$(HOST)' 'read -r k; cd /tmp && \
	  runuser -u vinyl -- env YT_DATA_API_KEY="$$k" VINYL_DATA_DIR=/var/lib/vinyl \
	  VINYL_SEED_FILE=/tmp/seed-channels.txt /opt/vinyl/vinyl ingest'

deploy: ## Build and deploy to the droplet
	deploy/deploy.sh '$(HOST)'
