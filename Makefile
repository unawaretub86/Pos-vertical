.PHONY: build frontend backend dev

frontend:
	cd frontend && npm run build
	go run scripts/copy-frontend.go

backend:
	go build -o pos.exe ./cmd/pos

build: frontend backend

dev-backend:
	go run ./cmd/pos

dev-frontend:
	cd frontend && npm run dev
