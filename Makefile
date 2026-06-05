PROTO_DIR := ./proto/
OUT_DIR := ./gen/go
OPENAPI_DIR := ./gen/openapiv2

proto-gen:
	@mkdir -p $(OUT_DIR) $(OPENAPI_DIR)
	PATH="$$PATH:$$(go env GOPATH)/bin" protoc --proto_path=$(PROTO_DIR) --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
	                                          --go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
	                                          --grpc-gateway_out=$(OUT_DIR) --grpc-gateway_opt=paths=source_relative \
	                                          --openapiv2_out=$(OPENAPI_DIR) \
	                                          ./proto/sso/sso.proto

proto-clean:
	rm -rf $(OUT_DIR) $(OPENAPI_DIR)

proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
