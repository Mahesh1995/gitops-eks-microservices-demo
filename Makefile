# Convenience targets. Run `make help` to list them.
SERVICES := frontend product-catalog cart
REGISTRY ?= ghcr.io/OWNER          # override: make build REGISTRY=ghcr.io/mahesh1995
TAG      ?= dev

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: up
up: ## Run the whole app locally with Docker Compose
	docker compose up --build

.PHONY: down
down: ## Stop the local Docker Compose app
	docker compose down

.PHONY: build
build: ## Build all service images ($(REGISTRY)/<svc>:$(TAG))
	@for s in $(SERVICES); do \
		echo "==> building $$s"; \
		docker build -t $(REGISTRY)/$$s:$(TAG) src/$$s; \
	done

.PHONY: k8s-dev
k8s-dev: ## Deploy the dev overlay to the current kube-context
	kubectl apply -k kustomize/overlays/dev

.PHONY: k8s-prod
k8s-prod: ## Deploy the prod overlay to the current kube-context
	kubectl apply -k kustomize/overlays/prod

.PHONY: kustomize-check
kustomize-check: ## Render overlays without applying (validation)
	@echo "--- dev ---";  kubectl kustomize kustomize/overlays/dev  > /dev/null && echo OK
	@echo "--- prod ---"; kubectl kustomize kustomize/overlays/prod > /dev/null && echo OK

.PHONY: helm-lint
helm-lint: ## Lint the Helm chart
	helm lint helm-chart

.PHONY: tf-init tf-plan
tf-init: ## terraform init
	cd terraform && terraform init
tf-plan: ## terraform plan
	cd terraform && terraform plan
