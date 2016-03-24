.PHONY: help
.DEFAULT_GOAL := help

deps: ## Vendor go dependencies
	@echo "Vendoring dependencies"
	@go get -u github.com/kardianos/govendor
	@govendor sync

install-dev: deps ## Install dependencies and prepared development configuration
	@echo "Installing development utils"

run-dev: ## Run the sample test program with a Consul backend
	@echo "Running consul on port 8500"
	@docker-compose up -d tconsul

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

