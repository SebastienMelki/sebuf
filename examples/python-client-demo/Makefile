PROTO_DIR := proto
OUT_DIR := gen

.PHONY: generate
generate:
	@mkdir -p $(OUT_DIR)
	@protoc \
		--plugin=protoc-gen-py-client=../../bin/protoc-gen-py-client \
		--proto_path=$(PROTO_DIR) \
		--proto_path=../../proto \
		--py-client_out=$(OUT_DIR) \
		--py-client_opt=package=demo/contracts/v1 \
		$(PROTO_DIR)/contracts.proto
	@echo "Generated Python client into $(OUT_DIR)/"

.PHONY: clean
clean:
	@rm -rf $(OUT_DIR)
