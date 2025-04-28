

## 
##-----------------------------------------------------------------------
## Rules for building the FOKS client -- the most common operations!
##

.PHONY: client
client:
	(cd client/foks && CGO_ENABLED=1 go install)
	@echo "Client binary is ready: $$(scripts/gowhere.sh)/foks"

client-proto: proto client
full: client-proto

.PHONY: client-linux-arm64
client-linux-arm64: build/foks.linux-arm64
	@echo "Client binary for linux/arm64 is ready: $<"

.PHONY: client-linux-amd64
client-linux-amd64: build/foks.linux-amd64
	@echo "Client binary for linux/amd64 is ready: $<"

.PHONY: git-link
git-link:
	./scripts/git-link.sh

.PHONY: ci
ci:
	bash ci.bash

.PHONY: ci-yubi-destructive
ci-yubi-destructive:
	bash ci.bash --yubi-destructive

.PHONY: macos-sign
macos-sign:
	codesign --deep \
         --options runtime \
         --timestamp \
         --sign "Developer ID Application: NE43 INC (L2W77ZPF94)" \
		 $$(scripts/gowhere.sh)/foks

.PHONY: macos-verify
macos-verify:
	codesign --verify --deep --strict --verbose=2 $$(scripts/gowhere.sh)/foks
	codesign -dvv $$(scripts/gowhere.sh)/foks

##
##-----------------------------------------------------------------------
##

.stamps/npm-install:
	npm i
	mkdir -p .stamps
	date > .stamps/npm-install

.PHONY: proto
proto: .stamps/npm-install
	(cd proto-src && go run ../tools/snowp-checker)
	go generate ./...

build/foks.linux-arm64: proto
	./scripts/cross-compile.sh linux-arm64

build/foks.linux-amd64: proto
	./scripts/cross-compile.sh linux-amd64