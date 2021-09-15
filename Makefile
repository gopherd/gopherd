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

define build_protobuf
	@echo Compiling proto/protobuf/$(1)/*.proto ...
	@protoc --gopherd_out=. proto/protobuf/$(1)/*.proto
endef

.PHONY: all
all: autogen cmd

.PHONY: autogen
autogen: proto auth/api

.PHONY: proto
proto:
	$(call build_protobuf,gatepb)

.PHONY: auth/api
auth/api:
	midc -Ogo=auth/ -Tgo=mid/templates/auth/ mid/auth.mid

.PHONY: lint
lint:
	@echo "Linting ./..."
	@go vet ./...
	@loglint ./...

.PHONY: cmd
cmd: lint gated authd

.PHONY: gated
gated:
	$(call build_target,gated)

.PHONY: authd
authd:
	$(call build_target,authd)
