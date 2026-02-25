# KrakenD Gateway Example Makefile
#
# Workflow: generate -> partials -> validate -> compose
#
# - generate: Run protoc with protoc-gen-krakend to produce per-service .krakend.json
# - partials: Extract endpoint arrays from generated files for Flexible Config
# - validate: Run krakend check -lc on each per-service config independently
# - compose:  Compose per-service fragments into a complete krakend.json via FC
# - all:      Full workflow (generate -> partials -> validate -> compose)
# - clean:    Remove all generated files

.PHONY: all generate partials validate compose clean

# Path to the krakend plugin binary (built from repo root with `make build`).
PLUGIN := ../../bin/protoc-gen-krakend

# Proto paths: local protos under proto/, plus sebuf annotations from repo root.
PROTO_PATH_LOCAL := proto
PROTO_PATH_SEBUF := ../../proto

# Proto source files.
PROTOS := proto/services/user_service.proto proto/services/product_service.proto

# Full workflow: generate, extract partials, validate per-service, compose gateway config.
all: generate partials validate compose
	@echo "Done. All configs generated and validated."

# Generate per-service KrakenD JSON configs from proto annotations.
# Uses protoc directly with --proto_path for local sebuf proto resolution.
# Output: generated/UserService.krakend.json, generated/ProductService.krakend.json
generate:
	@echo "Generating KrakenD configs..."
	@rm -rf generated
	@mkdir -p generated
	@protoc \
		--plugin=protoc-gen-krakend=$(PLUGIN) \
		--krakend_out=./generated \
		--proto_path=$(PROTO_PATH_LOCAL) \
		--proto_path=$(PROTO_PATH_SEBUF) \
		$(PROTOS)
	@echo "Generated:"
	@ls -1 generated/*.krakend.json

# Extract endpoint arrays from generated per-service files into partials.
# The partials strip the outer JSON envelope ($schema, version, endpoints wrapper)
# and produce bare array contents suitable for inclusion in the Flexible Config template.
#
# jq '.endpoints' extracts the endpoints array, then sed strips the outer [ and ]
# so the template can concatenate multiple services into a single endpoints array.
partials:
	@mkdir -p gateway/partials
	@echo "Extracting partials..."
	@jq '.endpoints' generated/UserService.krakend.json | sed '1d;$$d' > gateway/partials/user_endpoints.json
	@jq '.endpoints' generated/ProductService.krakend.json | sed '1d;$$d' > gateway/partials/product_endpoints.json
	@echo "Partials created:"
	@ls -1 gateway/partials/*.json

# Validate each per-service KrakenD config independently.
# krakend check -lc validates against the official KrakenD JSON schema.
# Per-service files are standalone -- they include $schema and version.
validate:
	@echo "Validating per-service configs..."
	@krakend check -l -c generated/UserService.krakend.json
	@krakend check -l -c generated/ProductService.krakend.json
	@echo "All configs valid."

# Compose a complete gateway config from per-service partials using Flexible Config.
#
# Environment variables:
#   FC_ENABLE=1         -- Enable Flexible Config template processing
#   FC_PARTIALS         -- Directory containing partial files for {{ include }}
#   FC_SETTINGS         -- Directory containing settings JSON files for {{ marshal }}
#
# The template (gateway/krakend.tmpl) uses {{ include }} to pull in endpoint
# arrays from each service partial, producing a single combined krakend.json.
compose:
	@echo "Composing gateway config with Flexible Config..."
	@FC_ENABLE=1 \
		FC_PARTIALS=gateway/partials \
		FC_SETTINGS=gateway/settings \
		krakend check -l -c gateway/krakend.tmpl
	@echo "Composed config validated. To inspect:"
	@echo "  FC_ENABLE=1 FC_PARTIALS=gateway/partials FC_SETTINGS=gateway/settings krakend check -d -c gateway/krakend.tmpl"

# Remove all generated files and extracted partials.
clean:
	@rm -rf generated
	@rm -rf gateway/partials
	@echo "Cleaned."
