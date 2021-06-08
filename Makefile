PKG=github.com/gopherd/doge/build
BRANCH=$(shell git symbolic-ref --short HEAD)
HASH=$(shell git rev-parse HEAD)
DATE=$(shell date "+%Y/%m/%d")
TIME=$(shell date "+%H:%M:%S")
GOBUILD=go build -ldflags "-X ${PKG}.branch=${BRANCH} -X ${PKG}.hash=${HASH} -X ${PKG}.date=${DATE} -X ${PKG}.time=${TIME}"
TARGET_DIR=./target

define build_target
	@mkdir -p ${TARGET_DIR}
	@echo "Building ${TARGET_DIR}/$(1) ..."
	@${GOBUILD} -o ${TARGET_DIR}/$(1) ./cmd/$(1)/
endef

define install_target
	@echo "Installing $(1) ..."
	@go install ./cmd/$(1)/
endef

define build_protobuf
	protoc --go_out=. --gopherd_out=. proto/protobuf/$(1)
endef

.PHONY: all
all: autogen cmd

.PHONY: install
install:
	$(call install_target,protoc-gen-gopherd)

.PHONY: autogen
autogen: proto

.PHONY: proto
proto:
	$(call build_protobuf,gatec.proto)
	$(call build_protobuf,gated.proto)

.PHONY: cmd
cmd: gated

.PHONY: gated
gated:
	$(call build_target,gated)

.PHONY: protoc-gen-gopherd
protoc-gen-gopherd:
	$(call build_target,protoc-gen-gopherd)
