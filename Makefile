.PHONY: help install backend frontend build docker-build docker-run k8s-apply

IMAGE ?= central-devtron:latest

help:
	@echo "Central-Devtron"
	@echo "  make install       - install frontend deps"
	@echo "  make backend       - run Go API + SQLite (writes ./central.db)"
	@echo "  make frontend      - run Vite dev server (proxies /api to backend)"
	@echo "  make docker-build  - build the single-container image ($(IMAGE))"
	@echo "  make docker-run    - run the full app locally via docker compose"
	@echo "  make k8s-apply     - kubectl apply -k k8s/"

install:
	cd frontend && npm install

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker compose up --build

k8s-apply:
	kubectl apply -k k8s/
