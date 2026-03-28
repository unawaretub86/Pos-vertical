.PHONY: build frontend backend dev icon install-tools build-windows

# Install dev tools (run once)
install-tools:
	go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

# Generate Windows icon resource (run once after install-tools, then commit resource.syso)
# Only needed if you want the .ico embedded in pos.exe
icon:
	cd cmd/pos && go generate

frontend:
	cd frontend && npm run build
	go run scripts/copy-frontend.go

backend:
	go build -ldflags="-s -w" -o pos.exe ./cmd/pos

build: frontend backend

# Cross-compile for Windows from any OS (no CGO required - uses modernc.org/sqlite)
build-windows:
	cd frontend && npm run build
	go run scripts/copy-frontend.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o pos.exe ./cmd/pos

dev-backend:
	go run ./cmd/pos

dev-frontend:
	cd frontend && npm run dev
