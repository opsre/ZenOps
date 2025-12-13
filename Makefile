# 定义项目名称
BINARY_NAME=zenops

# 定义输出目录
OUTPUT_DIR=bin

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
	@echo "  run         运行项目"
	@echo "  build       为当前平台构建可执行文件"
	@echo "  gox-linux   为Linux平台构建可执行文件"
	@echo "  gox-all     为所有平台构建可执行文件"
	@echo "  clean       清理生成的可执行文件"
	@echo "  lint        代码格式检查"
	@echo "  help        显示帮助信息"