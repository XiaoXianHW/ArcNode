.PHONY: all build frontend gateway dev clean

all: build

build: frontend gateway

frontend:
	cd frontend && npm install --no-audit --no-fund && npm run build

gateway:
	cd gateway && go build -o ../arcnode-gateway .

dev:
	cd gateway && go run .

clean:
	rm -rf frontend/node_modules frontend/dist gateway/web/dist/*
	mkdir -p gateway/web/dist
	touch gateway/web/dist/.gitkeep
	rm -f arcnode-gateway
