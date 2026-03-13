.PHONY: proto-build proto-go proto-ts proto-clean

# Protobuf compilation
PROTOC := protoc
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GRPC_WEB := $(shell go env GOPATH)/bin/protoc-gen-grpc-web
PROTOC_GEN_TS := $(shell go env GOPATH)/bin/protoc-gen-ts

# Directories
PROTO_DIR := proto
GO_OUT_DIR := proto
TS_OUT_DIR := ../web/src/proto

# Go module path
GO_MODULE := wws/api/proto

# Build all protobuf files
proto-build: proto-go proto-ts

# Generate Go code from protobuf
proto-go: $(PROTOC_GEN_GO)
	$(PROTOC) \
		--go_out=$(GO_OUT_DIR) \
		--go_opt=module=$(GO_MODULE) \
		-I$(PROTO_DIR) \
		$(PROTO_DIR)/common/common.proto \
		$(PROTO_DIR)/user/user.proto \
		$(PROTO_DIR)/auth/auth.proto \
		$(PROTO_DIR)/organization/organization.proto \
		$(PROTO_DIR)/workspace/workspace.proto

# Generate TypeScript code from protobuf
proto-ts: $(PROTOC_GEN_GRPC_WEB) $(PROTOC_GEN_TS)
	# Generate TypeScript code
	$(PROTOC) \
		--plugin=protoc-gen-ts=$(PROTOC_GEN_TS) \
		--ts_out=$(TS_OUT_DIR) \
		--ts_opt=esm=true \
		-I$(PROTO_DIR) \
		$(PROTO_DIR)/common/common.proto \
		$(PROTO_DIR)/user/user.proto \
		$(PROTO_DIR)/auth/auth.proto \
		$(PROTO_DIR)/organization/organization.proto \
		$(PROTO_DIR)/workspace/workspace.proto
	
	# Generate gRPC-Web code
	$(PROTOC) \
		--plugin=protoc-gen-grpc-web=$(PROTOC_GEN_GRPC_WEB) \
		--grpc-web_out=import_style=typescript,mode=grpcweb:$(TS_OUT_DIR) \
		-I$(PROTO_DIR) \
		$(PROTO_DIR)/auth/auth.proto \
		$(PROTO_DIR)/organization/organization.proto \
		$(PROTO_DIR)/workspace/workspace.proto

# Install protobuf tools
proto-tools:
	# Install protoc-gen-go
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	
	# Install protoc-gen-grpc-web
	@echo "Installing protoc-gen-grpc-web..."
	@npm install -g protoc-gen-grpc-web || \
		curl -L https://github.com/grpc/grpc-web/releases/download/1.4.2/protoc-gen-grpc-web-1.4.2-linux-x86_64 -o $(shell go env GOPATH)/bin/protoc-gen-grpc-web && \
		chmod +x $(shell go env GOPATH)/bin/protoc-gen-grpc-web
	
	# Install protoc-gen-ts
	@echo "Installing protoc-gen-ts..."
	@npm install -g protoc-gen-ts || echo "Manual installation may be required"

# Install protoc if not present
install-protoc:
	@echo "Checking for protoc..."
	@which protoc > /dev/null || (echo "protoc not found. Installing..." && \
		sudo apt-get update && \
		sudo apt-get install -y protobuf-compiler)

# Clean generated files
proto-clean:
	rm -f $(GO_OUT_DIR)/*/*.pb.go
	rm -rf $(TS_OUT_DIR)/*

# Validate protobuf files
proto-lint:
	@which buf > /dev/null && buf lint || echo "buf not installed. Install with: go install github.com/bufbuild/buf/cmd/buf@latest"

# Check protobuf compilation
proto-check: proto-build
	@echo "Protobuf compilation successful!"
	@ls -la $(GO_OUT_DIR)/*/*.pb.go
