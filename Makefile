VERSION=$(shell git describe 2>/dev/null || echo "develop")
BUILD=$(shell git rev-parse --short HEAD 2>/dev/null || echo "develop")
PLATFORMS := windows linux darwin
GOOS = $(word 1, $@)
BINARY := turing-screen
LDFLAGS=-ldflags "-s -w -X=github.com/alexwbaule/turing-screen/internal/application/logger.Version=$(VERSION) -X=github.com/alexwbaule/turing-screen/internal/application/logger.Build=$(BUILD)"

INSTALL_DIR   := /opt/smart-screen
SYSTEMD_DIR   := /etc/systemd/system
DESKTOP_DIR   := /usr/share/applications

LOCALE_DIR    := interface/locale
PY_SOURCES    := $(wildcard interface/ui/*.py) interface/main.py interface/i18n.py
PO_FILES      := $(wildcard $(LOCALE_DIR)/*/LC_MESSAGES/turing-screen.po)
MO_FILES      := $(PO_FILES:.po=.mo)


build:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/$(BINARY) -v cmd/$(BINARY)/main.go

build-sensors-test:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/turing-test-sensors -v cmd/turing-test-sensors/main.go

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

build-all: build build-test build-sensors-test

# ── i18n ─────────────────────────────────────────────────────────────────────

# Compile every PO file to its MO counterpart.
.PHONY: locale
locale: $(MO_FILES)

%.mo: %.po
	msgfmt $< -o $@

# Extract translatable strings from the Python interface into the POT template,
# then merge new strings into each existing PO file.
# Run this whenever new _("…") calls are added to the source.
# After running, edit the PO files and run 'make locale' to recompile.
.PHONY: locale-update
locale-update:
	@echo "==> Extracting strings from Python interface..."
	xgettext --language=Python --keyword=_ --from-code=UTF-8 \
	         --package-name=turing-screen \
	         --output=$(LOCALE_DIR)/turing-screen.pot \
	         $(PY_SOURCES)
	@echo "==> Merging into existing PO files..."
	@for po in $(PO_FILES); do \
		echo "  $$po"; \
		msgmerge --update --no-fuzzy-matching --quiet $$po $(LOCALE_DIR)/turing-screen.pot; \
	done
	@echo "==> Done. Edit PO files then run: make locale"

GROUP := smart-screen

.PHONY: install
install: build-all locale
	@echo "==> Creating group $(GROUP)"
	getent group $(GROUP) >/dev/null || groupadd --system $(GROUP)
	@echo "==> Installing to $(INSTALL_DIR)"
	install -d $(INSTALL_DIR)/bin
	install -d -m 775 -g $(GROUP) $(INSTALL_DIR)/conf
	install -m 755 bin/$(BINARY)        $(INSTALL_DIR)/bin/$(BINARY)
	install -m 755 bin/turing-interface $(INSTALL_DIR)/bin/turing-interface
	@# Python interface: i18n bootstrap + compiled locale catalogs
	install -d $(INSTALL_DIR)/interface
	install -m 644 interface/i18n.py $(INSTALL_DIR)/interface/i18n.py
	cp -r $(LOCALE_DIR) $(INSTALL_DIR)/interface/locale
	find $(INSTALL_DIR)/interface/locale -name '*.po' -delete
	@# Copy res/ preserving existing files on reinstall
	cp -rn res/ $(INSTALL_DIR)/
	@# Group smart-screen owns res/; dirs 775, files 664
	chown -R root:$(GROUP) $(INSTALL_DIR)/res
	find $(INSTALL_DIR)/res -type d -exec chmod 775 {} +
	find $(INSTALL_DIR)/res -type f -exec chmod 664 {} +
	@# Default config — skip if already present (preserve user edits on reinstall)
	@if [ ! -f $(INSTALL_DIR)/conf/config.yaml ]; then \
		install -m 664 conf/config.yaml $(INSTALL_DIR)/conf/config.yaml; \
		echo "  Installed default config.yaml"; \
	else \
		echo "  Keeping existing conf/config.yaml"; \
	fi
	chown root:$(GROUP) $(INSTALL_DIR)/conf
	chown root:$(GROUP) $(INSTALL_DIR)/conf/config.yaml
	chmod 664 $(INSTALL_DIR)/conf/config.yaml
	@echo "==> Installing systemd services"
	install -m 644 scripts/smart-screen-go.service          $(SYSTEMD_DIR)/smart-screen-go.service
	install -m 644 scripts/sleep@smart-screen-go.service    $(SYSTEMD_DIR)/sleep@smart-screen-go.service
	systemctl daemon-reload
	@echo "==> Installing icon and desktop entry"
	install -Dm644 res/icon.svg /usr/share/icons/hicolor/scalable/apps/turing-screen.svg
	install -m 644 scripts/turing-interface.desktop $(DESKTOP_DIR)/turing-interface.desktop
	@# Auto-add the invoking user (SUDO_USER) to the group
	@if [ -n "$$SUDO_USER" ]; then \
		if ! id -nG "$$SUDO_USER" 2>/dev/null | grep -qw "$(GROUP)"; then \
			usermod -aG $(GROUP) $$SUDO_USER; \
			echo "==> Usuário '$$SUDO_USER' adicionado ao grupo '$(GROUP)'."; \
			echo "    Faça re-login ou execute: newgrp $(GROUP)"; \
		else \
			echo "==> Usuário '$$SUDO_USER' já está no grupo '$(GROUP)'."; \
		fi \
	else \
		echo "==> Aviso: SUDO_USER não detectado. Adicione seu usuário manualmente:"; \
		echo "    sudo usermod -aG $(GROUP) $$USER"; \
	fi
	@echo ""
	@echo "To enable the daemon on boot:"
	@echo "  sudo systemctl enable --now smart-screen-go"
	@echo "  sudo systemctl enable sleep@smart-screen-go"
