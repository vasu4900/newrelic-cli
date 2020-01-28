protobuf_folder ?= ./internal/plugins/protoDef

# grpc-tools must be installed to run this target.
# brew install bradleyjkemp/formulae/grpc-tools
protobuf:
	@echo "=== $(PROJECT_NAME) === [ protobuf         ]: generating protobuf contract."
	@protoc -I=$(protobuf_folder) --go_out=plugins=grpc:$(protobuf_folder) ${protobuf_folder}/cli-plugin.proto

.phony: protobuf
