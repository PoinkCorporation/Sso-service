PROTO_DIR := ./proto/
OUT_DIR := ./gen/go

proto-gen:
	@mkdir -p $(OUT_DIR)
	set PATH "$PATH:$(go env GOPATH)/bin"
	protoc --proto_path=$(PROTO_DIR) --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
                                          --go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
                                          ./proto/sso/sso.proto
proto-clean:
	rm -f $(OUT_DIR)/*.pb.go $(OUT_DIR)/*_grpc.pb.go
proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest