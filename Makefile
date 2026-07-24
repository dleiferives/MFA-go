# MFA-go: Go wrapper for Montreal Forced Aligner

.PHONY: setup test build clean

MAMBA_URL ?= https://github.com/mamba-org/micromamba-releases/releases/download/2.0.10-0/micromamba-linux-64
ENV_DIR    ?= mfa/env

build:
	@go build ./...
	@echo "  → built"

test:
	@go test ./...
	@echo "  → tests passed"

setup-micromamba:
	@mkdir -p mfa/bin
	@if [ ! -x mfa/bin/micromamba ]; then \
		echo "  → downloading micromamba..."; \
		curl -sSL $(MAMBA_URL) -o mfa/bin/micromamba; \
		chmod +x mfa/bin/micromamba; \
	fi
	@echo "  → micromamba ready"

setup-env: setup-micromamba
	@if [ ! -d $(ENV_DIR)/bin/mfa ]; then \
		echo "  → creating MFA environment (this will take a while)..."; \
		mfa/bin/micromamba create -y -p $(ENV_DIR) -c conda-forge \
			montreal-forced-aligner 'kaldi=*=cpu*'; \
	fi
	@echo "  → MFA environment ready"

setup-models: setup-env
	@export PATH="$(ENV_DIR)/bin:$$PATH" && \
	mkdir -p $(ENV_DIR)/work/pretrained_models && \
	mfa model download acoustic greek_cv && \
	mfa model download dictionary greek_cv && \
	mfa model download acoustic english_mfa && \
	mfa model download dictionary english_us_arpa
	@echo "  → models downloaded"

setup: setup-models
	@echo "  → MFA-go setup complete"

clean:
	rm -rf mfa/env mfa/work mfa/.mamba mfa/bin/micromamba
	@echo "  → cleaned"
