.PHONY: build build-frontend run clean dev backend

# 构建前端
build-frontend:
	cd web && npm install && npm run build

# 构建后端（包含嵌入的前端）
build: build-frontend
	go build -o codex-agent-team ./cmd/server

# 运行服务器
run: build
	./codex-agent-team -codex /home/catstream/.local/bin/codex2 -repo .

# 开发模式
dev:
	@echo "Terminal 1: cd web && npm run dev"
	@echo "Terminal 2: go run cmd/server/main.go -codex codex2 -repo ."

# 清理
clean:
	rm -f codex-agent-team
	rm -rf web/dist

# 仅构建后端
backend:
	go build -o codex-agent-team ./cmd/server
