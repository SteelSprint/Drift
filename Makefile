.PHONY: build eval

build:
	go build -o drift ./cmd/drift

eval: build
	@if [ -z "$(PROMPT)" ]; then \
		echo "Usage: make eval PROMPT=\"<your prompt>\""; \
		echo "  e.g. make eval PROMPT=\"create a working CLI version of poker\""; \
		echo ""; \
		echo "Or run the full test battery:"; \
		echo "  go run ./eval --battery"; \
		echo ""; \
		echo "Other eval options:"; \
		echo "  go run ./eval --dry-run \"<prompt>\"     # stage only, skip LLM calls"; \
		echo "  go run ./eval --subject <model> \"<prompt>\"  # override subject model"; \
		echo "  go run ./eval --judge <model> \"<prompt>\"    # override judge model"; \
		echo "  go run ./eval --label <name> \"<prompt>\"     # name the run"; \
		exit 1; \
	fi
	@go run ./eval "$(PROMPT)"
