
GO_BUILD_ENV :=
# -s -w drops symbols and DWARF, replacing a host `strip` that cannot handle cross-built binaries.
GO_BUILD_FLAGS := -ldflags "-s -w"
MODULE_BINARY := bin/system-vitals
# One custom control per model; each builds to <dir>/dist/index.html and is declared in meta.json.
UI_ARTIFACTS := vitals_ui/dist/index.html

ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS += -tags no_cgo
	MODULE_BINARY = bin/system-vitals.exe
endif

$(MODULE_BINARY): Makefile go.mod *.go cmd/module/*.go internal/*/*.go
	GOOS=$(VIAM_BUILD_OS) GOARCH=$(VIAM_BUILD_ARCH) $(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) cmd/module/main.go

# Builds a control UI whenever its sources change. Needs node and pnpm.
.SECONDEXPANSION:
%/dist/index.html: $$(wildcard $$*/src/*) $$(wildcard $$*/src/*/*) $$*/index.html $$*/package.json $$*/vite.config.ts
	cd $* && pnpm install --frozen-lockfile && pnpm build

ui: $(UI_ARTIFACTS)

lint:
	gofmt -s -w .

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	go test ./...

FIRST_RUN := $(shell jq -r '.first_run // empty' meta.json 2>/dev/null)
TAR_FILES := meta.json $(MODULE_BINARY) $(UI_ARTIFACTS)
ifneq ($(FIRST_RUN),)
TAR_FILES += $(FIRST_RUN)
endif

module.tar.gz: meta.json $(MODULE_BINARY) $(UI_ARTIFACTS)
	tar czf $@ $(TAR_FILES)

module: test module.tar.gz

all: test module.tar.gz

setup:
	./setup.sh

clean:
	rm -f module.tar.gz $(MODULE_BINARY)
	rm -rf $(dir $(UI_ARTIFACTS))

.PHONY: lint update test module all setup ui clean
