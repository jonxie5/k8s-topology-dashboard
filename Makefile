.PHONY: api-run api-build ui-dev ui-build build

## Run the API locally (uses $KUBECONFIG or ~/.kube/config)
api-run:
	cd api && go run ./cmd/dashboard-api \
		--cors-origin http://localhost:5173

## Build the API binary to api/bin/dashboard-api
api-build:
	cd api && go build -o bin/dashboard-api ./cmd/dashboard-api

## Start the Vite dev server for the UI
ui-dev:
	cd ui && npm install && npm run dev

## Build the UI for production (output: ui/dist/)
ui-build:
	cd ui && npm install && npm run build

## Build both services
build: api-build ui-build
