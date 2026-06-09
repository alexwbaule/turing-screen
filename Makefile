VERSION=$(shell git describe 2>/dev/null || echo "develop")
BUILD=$(shell git rev-parse --short HEAD 2>/dev/null || echo "develop")
PLATFORMS := windows linux darwin
GOOS = $(word 1, $@)
BINARY := turing-screen
LDFLAGS=-ldflags "-s -w -X=github.com/alexwbaule/turing-screen/internal/application/logger.Version=$(VERSION) -X=github.com/alexwbaule/turing-screen/internal/application/logger.Build=$(BUILD)"

INSTALL_DIR   := /opt/smart-screen
SYSTEMD_DIR   := /etc/systemd/system
DESKTOP_DIR   := /usr/share/applications


build:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/$(BINARY) -v cmd/$(BINARY)/main.go

build-sensors-test:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/turing-test-sensors -v cmd/turing-test-sensors/main.go

build-editor:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/turing-interface -v cmd/turing-interface/main.go

build-test:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/turing-test -v cmd/turing-test/main.go

.PHONY: $(PLATFORMS)

$(PLATFORMS): build
	mkdir -p bin/
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-$(GOOS)-amd64 cmd/$(BINARY)/main.go
	chmod 755 bin/$(BINARY)-$(GOOS)-amd64

.PHONY: release
release: windows linux darwin

build-all: build build-editor build-test build-sensors-test

GROUP := smart-screen

.PHONY: install
install: build-all
	@echo "==> Creating group $(GROUP)"
	getent group $(GROUP) >/dev/null || groupadd --system $(GROUP)
	@echo "==> Installing to $(INSTALL_DIR)"
	install -d $(INSTALL_DIR)/bin
	install -d -m 775 -g $(GROUP) $(INSTALL_DIR)/conf
	install -m 755 bin/$(BINARY)        $(INSTALL_DIR)/bin/$(BINARY)
	install -m 755 bin/turing-interface $(INSTALL_DIR)/bin/turing-interface
	@# Copy res/ preserving existing files on reinstall
	cp -rn res/ $(INSTALL_DIR)/
	@# Group smart-screen owns res/; dirs 775, files 664
	chown -R root:$(GROUP) $(INSTALL_DIR)/res
	find $(INSTALL_DIR)/res -type d -exec chmod 775 {} +
	find $(INSTALL_DIR)/res -type f -exec chmod 664 {} +
	@# Default config — skip if already present (preserve user edits on reinstall)
	@if [ ! -f $(INSTALL_DIR)/conf/config.yaml ]; then \
		install -m 664 -g $(GROUP) conf/config.yaml $(INSTALL_DIR)/conf/config.yaml; \
		echo "  Installed default config.yaml"; \
	else \
		echo "  Keeping existing conf/config.yaml"; \
	fi
	chown root:$(GROUP) $(INSTALL_DIR)/conf
	@echo "==> Installing systemd services"
	install -m 644 scripts/smart-screen-go.service          $(SYSTEMD_DIR)/smart-screen-go.service
	install -m 644 scripts/sleep@smart-screen-go.service    $(SYSTEMD_DIR)/sleep@smart-screen-go.service
	systemctl daemon-reload
	@echo "==> Installing icon and desktop entry"
	install -Dm644 res/icon.svg /usr/share/icons/hicolor/scalable/apps/turing-screen.svg
	install -m 644 scripts/turing-interface.desktop $(DESKTOP_DIR)/turing-interface.desktop
	@echo ""
	@echo "Done. Add your user to the $(GROUP) group:"
	@echo "  sudo usermod -aG $(GROUP) $$USER  (re-login to take effect)"
	@echo ""
	@echo "To enable the daemon on boot:"
	@echo "  sudo systemctl enable --now smart-screen-go"
	@echo "  sudo systemctl enable sleep@smart-screen-go"
