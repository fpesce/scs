export PATH := /usr/local/go/bin:/home/joke/go/bin:$(PATH)

.PHONY: lint test e2etest

lint:
	golangci-lint run ./...

test:
	CGO_ENABLED=1 go test -v -race ./...

e2etest:
	@for f in e2etests/*.txt; do \
		echo "========================================"; \
		echo "Running scs on $$f..."; \
		start=$$(date +%s%3N); \
		./scs build -i "$$f" -o "$${f%.txt}.scs"; \
		end=$$(date +%s%3N); \
		elapsed=$$((end-start)); \
		txt_size=$$(wc -c < "$$f"); \
		scs_size=$$(wc -c < "$${f%.txt}.scs"); \
		if [ $$txt_size -eq 0 ]; then \
			ratio=0; \
		else \
			ratio=$$(awk "BEGIN {printf \"%.2f\", $$scs_size / $$txt_size * 100}"); \
		fi; \
		echo "Execution Time: $${elapsed}ms"; \
		echo "Size: $$txt_size B -> $$scs_size B ($$ratio%)"; \
	done
