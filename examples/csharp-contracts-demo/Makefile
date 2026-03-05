PROTO_DIR := proto
OUT_DIR := gen

.PHONY: generate
generate:
	@mkdir -p $(OUT_DIR)/newtonsoft $(OUT_DIR)/system-text-json
	@protoc \
		--plugin=protoc-gen-csharp-http=../../bin/protoc-gen-csharp-http \
		--proto_path=$(PROTO_DIR) \
		--proto_path=../../proto \
		--csharp-http_out=$(OUT_DIR)/newtonsoft \
		--csharp-http_opt=namespace=Demo.Contracts,json_lib=newtonsoft \
		$(PROTO_DIR)/contracts.proto
	@protoc \
		--plugin=protoc-gen-csharp-http=../../bin/protoc-gen-csharp-http \
		--proto_path=$(PROTO_DIR) \
		--proto_path=../../proto \
		--csharp-http_out=$(OUT_DIR)/system-text-json \
		--csharp-http_opt=namespace=Demo.Contracts,json_lib=system_text_json \
		$(PROTO_DIR)/contracts.proto
	@echo "Generated C# contracts into $(OUT_DIR)/"

.PHONY: clean
clean:
	@rm -rf $(OUT_DIR)
