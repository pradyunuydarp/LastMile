# LastMile developer + demo commands.  `make help` lists them.

IMAGE_PREFIX ?= pradyunuydarp/lastmile
TAG          ?= latest
DOCKER       ?= docker
NS           ?= lastmile
SERVICES     := station driver rider trip notification location user matching gateway loadgen

.PHONY: help proto test race lint run-local images web-image compose-up compose-down \
        k8s-up k8s-down k8s-status scale-up scale-down chaos load-start load-stop

help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ---- develop ---------------------------------------------------------------
proto: ## Regenerate gen/go from api/*.proto (needs protoc + protoc-gen-go[-grpc])
	protoc -I . --go_out=module=lastmile:. --go-grpc_out=module=lastmile:. api/*.proto

test: ## Run all Go tests
	go test ./...

race: ## Run all Go tests with the race detector (what CI should run)
	go test -race -count=1 ./...

lint: ## gofmt + go vet
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; echo "run: gofmt -w cmd internal"; exit 1)
	go vet ./...

run-local: ## Run every service as a local process (no Docker)
	scripts/run-local.sh

# ---- containers ------------------------------------------------------------
images: ## Build every Go service image
	@for s in $(SERVICES); do \
		echo "▸ $$s"; $(DOCKER) build -q --build-arg SERVICE=$$s -t $(IMAGE_PREFIX)-$$s:$(TAG) . >/dev/null || exit 1; \
	done

web-image: ## Build the web app image
	$(DOCKER) build -t $(IMAGE_PREFIX)-web:$(TAG) web

compose-up: ## Whole system in Docker with 3 Matching replicas (web on :8080)
	$(DOCKER) compose up --build -d --scale matching=3

compose-down: ## Stop the Docker system
	$(DOCKER) compose --profile load down

# ---- Kubernetes demo (minikube) -----------------------------------------------
k8s-up: ## Build images inside minikube and deploy everything
	eval "$$(minikube docker-env)" && $(MAKE) images web-image
	minikube addons enable metrics-server
	kubectl apply -f k8s/lastmile.yaml -f k8s/web.yaml
	kubectl -n $(NS) rollout status deploy/gateway --timeout=120s

k8s-down: ## Delete the whole namespace
	kubectl delete namespace $(NS)

k8s-status: ## Pods, HPA, and the gateway's view of service health
	kubectl -n $(NS) get pods,hpa
	@kubectl -n $(NS) run status-probe --rm -i --restart=Never --image=curlimages/curl -q -- \
		curl -s gateway/system/status || true

scale-up: ## Manually scale Matching to 5 replicas
	kubectl -n $(NS) scale deploy/matching --replicas=5

scale-down: ## Back to 1 Matching replica
	kubectl -n $(NS) scale deploy/matching --replicas=1

chaos: ## Kill one random Matching pod (resilience demo)
	kubectl -n $(NS) delete $$(kubectl -n $(NS) get pods -l app=matching -o name | shuf -n1) --wait=false

load-start: ## Start the load generator (HPA demo); watch with: kubectl -n lastmile logs -f deploy/loadgen
	kubectl apply -f k8s/loadgen.yaml

load-stop: ## Stop the load generator
	kubectl delete -f k8s/loadgen.yaml --ignore-not-found
