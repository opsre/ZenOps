# 定义项目名称
BINARY_NAME=zenops

# 定义输出目录
OUTPUT_DIR=bin

# 前端目录
WEB_DIR=zenops-web

VERSION    = $(shell git describe --tags --always)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date "+%F %T")

define LDFLAGS
"-X 'github.com/eryajf/zenops/cmd.Version=${VERSION}' \
 -X 'github.com/eryajf/zenops/cmd.GitCommit=${GIT_COMMIT}' \
 -X 'github.com/eryajf/zenops/cmd.BuildTime=${BUILD_TIME}'"
endef

.PHONY: default
default: help

.PHONY: run
run:
	go run -ldflags=${LDFLAGS} main.go run

.PHONY: build
build:
	go build -ldflags=${LDFLAGS} -o ${BINARY_NAME} main.go

# 构建前端
.PHONY: build-web
build-web:
	@echo ">>> 构建前端..."
	cd ${WEB_DIR} && npm install && npm run build
	@echo ">>> 复制前端产物到 web/dist..."
	rm -rf web/dist
	cp -r ${WEB_DIR}/dist web/dist
	@echo ">>> 前端构建完成"

# 构建全部（前端 + 后端）
.PHONY: build-all
build-all: build-web build
	@echo ">>> 前后端构建完成"

# 一键运行（构建全部并启动服务）
.PHONY: dev
dev: build-all
	@echo ">>> 启动服务..."
	./$(BINARY_NAME) run

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=${LDFLAGS} -o ${BINARY_NAME} main.go

.PHONY: lint
lint:
	env GOGC=25 golangci-lint run --fix -j 8 -v ./... --timeout=10m

.PHONY: gox-linux
gox-linux:
	CGO_ENABLED=0 gox -osarch="linux/amd64 linux/arm64" -ldflags=${LDFLAGS} -output="${OUTPUT_DIR}/${BINARY_NAME}_{{.OS}}_{{.Arch}}"
	@for b in $$(ls ${OUTPUT_DIR}); do \
		OUTPUT_FILE="${OUTPUT_DIR}/$$b"; \
		upx -9 -q "$$OUTPUT_FILE"; \
	done

.PHONY: gox-all
gox-all:
	CGO_ENABLED=0 gox -osarch="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/ppc64le windows/amd64" -ldflags=${LDFLAGS} -output="${OUTPUT_DIR}/${BINARY_NAME}_{{.OS}}_{{.Arch}}"
	@for b in $$(ls ${OUTPUT_DIR}); do \
		FILENAME=$$(basename -s .exe "$$b"); \
		GOOS=$$(echo "$$FILENAME" | rev | cut -d'_' -f2 | rev); \
		GOARCH=$$(echo "$$FILENAME" | rev | cut -d'_' -f1 | rev); \
		OUTPUT_FILE="${OUTPUT_DIR}/$$b"; \
		if [ "$$GOOS" = "windows" ] && [ "$$GOARCH" = "arm64" ]; then \
			echo "跳过 $$OUTPUT_FILE (Windows/arm64 不压缩)"; \
		elif [ "$$GOOS" = "darwin" ]; then \
			echo "压缩 macOS 文件: $$OUTPUT_FILE"; \
			upx --force-macos -fq -9 "$$OUTPUT_FILE"; \
		else \
			echo "压缩通用文件: $$OUTPUT_FILE"; \
			upx -q -9 "$$OUTPUT_FILE"; \
		fi; \
	done

.PHONY: clean
clean:
	@rm -rf ${OUTPUT_DIR}

# 帮助信息
.PHONY: help
help:
	@echo "参数:"
	@echo "  dev         一键构建前后端并运行服务"
	@echo "  run         运行项目（仅后端）"
	@echo "  build       为当前平台构建后端可执行文件"
	@echo "  build-web   构建前端"
	@echo "  build-all   构建前端 + 后端"
	@echo "  gox-linux   为Linux平台构建可执行文件"
	@echo "  gox-all     为所有平台构建可执行文件"
	@echo "  clean       清理生成的可执行文件"
	@echo "  lint        代码格式检查"
	@echo "  help        显示帮助信息"