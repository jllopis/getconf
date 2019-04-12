.PHONY: help
.DEFAULT_GOAL := help

run-dev: ## Run a Consul backend for testing
	@echo "Running consul on port 8500"
	@docker-compose -f docker-compose.yml up -d gconf.consul

stop-dev: ## Stop the Consul backend for testing
	@echo "Stopping consul instance"
	@docker-compose -f docker-compose.yml stop gconf.consul

# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

