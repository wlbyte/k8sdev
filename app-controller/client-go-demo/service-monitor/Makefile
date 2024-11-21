BINARY = resource-monitor
VERSION = $(shell git describe --tags --always)
GO = go
GOFLAGS = build -o $(BINARY)
REPOSITORY = registry.cn-hangzhou.aliyuncs.com/wlbyte
TAG := latest

export GO111MODULE := on

.PHONY: all
all: build

.PHONY: build
build: 
	$(GO) $(GOFLAGS)

.PHONY: clean
clean:
	rm -f $(BINARY)

.PHONY: run
run: build
	./$(BINARY)

.PHONY: docker
docker:
	docker buildx build -t $(REPOSITORY)/$(BINARY):$(TAG) . 
	
.PHONY: version
version:
	@echo $(version)