# Maintainer: Alex W Baulé <claude@abaule.me>
#
# Split package: build the Go daemon once, ship it as `turing-screen`, and ship
# the GTK4/Python editor as a separate `turing-interface` package that depends
# on it (the GUI shares the daemon's res/ and conf/ under /opt/smart-screen).
pkgbase=turing-screen
pkgname=('turing-screen' 'turing-interface')
pkgver=1.4.0.r4.g2890d3d
pkgrel=1
pkgdesc="Daemon and theme editor for Turing Smart Screen USB displays"
arch=('x86_64')
url="https://github.com/alexwbaule/turing-screen"
license=('custom')
makedepends=('go' 'gcc' 'git')
source=("$pkgbase::git+https://github.com/alexwbaule/turing-screen.git")
sha256sums=('SKIP')

pkgver() {
	cd "$pkgbase"
	git describe --tags --long 2>/dev/null | sed 's/\([^-]*-g\)/r\1/;s/-/./g' || \
		printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

prepare() {
	cd "$pkgbase"
	go mod download
}

build() {
	cd "$pkgbase"
	export CGO_ENABLED=1
	export GOFLAGS="-trimpath"
	export GOPATH="$srcdir/go"

	VERSION=$(git describe 2>/dev/null || echo "develop")
	BUILD=$(git rev-parse --short HEAD 2>/dev/null || echo "develop")
	LDFLAGS="-s -w \
		-X=github.com/alexwbaule/turing-screen/internal/application/logger.Version=$VERSION \
		-X=github.com/alexwbaule/turing-screen/internal/application/logger.Build=$BUILD"

	# Only the daemon is built from Go. The editor is the Python app in
	# interface/ (cmd/turing-interface was the old Fyne editor, now retired).
	go build -ldflags "$LDFLAGS" -o bin/turing-screen cmd/turing-screen/main.go
}

# -----------------------------------------------------------------------------
# turing-screen — the daemon (USB renderer + WebSocket API)
# -----------------------------------------------------------------------------

package_turing-screen() {
	pkgdesc="Daemon for Turing Smart Screen USB displays"
	depends=('gcc-libs')
	optdepends=('turing-interface: graphical theme editor and device manager')
	backup=('opt/smart-screen/conf/config.yaml')
	install=turing-screen.install

	cd "$pkgbase"

	# Daemon binary
	install -Dm755 bin/turing-screen "$pkgdir/opt/smart-screen/bin/turing-screen"

	# Resources: themes, fonts, icon
	# Owned by root:smart-screen; dirs 775, files 664 so group members can write
	cp -r res/ "$pkgdir/opt/smart-screen/"
	find "$pkgdir/opt/smart-screen/res" -type d -exec chmod 775 {} +
	find "$pkgdir/opt/smart-screen/res" -type f -exec chmod 664 {} +

	# Default config — listed in backup[] so pacman handles upgrades safely
	install -Dm644 conf/config.yaml "$pkgdir/opt/smart-screen/conf/config.yaml"
	chmod 775 "$pkgdir/opt/smart-screen/conf"

	# Systemd services
	install -Dm644 scripts/smart-screen-go.service \
		"$pkgdir/usr/lib/systemd/system/smart-screen-go.service"
	install -Dm644 "scripts/sleep@smart-screen-go.service" \
		"$pkgdir/usr/lib/systemd/system/sleep@smart-screen-go.service"
}

# -----------------------------------------------------------------------------
# turing-interface — the GTK4/Python theme editor and device manager
# -----------------------------------------------------------------------------

package_turing-interface() {
	pkgdesc="Theme editor and device manager GUI (GTK4) for Turing Smart Screen"
	depends=('turing-screen'
	         'python' 'python-gobject' 'gtk4' 'libadwaita' 'python-cairo'
	         'python-websockets' 'python-yaml' 'python-pillow')
	optdepends=('gnome-shell-extension-appindicator: system tray icon on GNOME')
	install=turing-interface.install

	# Cada package_*() começa em $srcdir; é preciso entrar no checkout.
	cd "$pkgbase"

	# Python sources live under /opt/smart-screen/interface/ so main.py's
	# ../conf/config.yaml and the cwd-based res/themes lookup both resolve
	# against the daemon's /opt/smart-screen tree.
	install -d "$pkgdir/opt/smart-screen/interface"
	install -Dm644 interface/main.py      "$pkgdir/opt/smart-screen/interface/main.py"
	install -Dm644 interface/ws_client.py  "$pkgdir/opt/smart-screen/interface/ws_client.py"
	cp -r interface/ui   "$pkgdir/opt/smart-screen/interface/ui"
	cp -r interface/theme "$pkgdir/opt/smart-screen/interface/theme"
	# Drop any cached bytecode from the working tree
	find "$pkgdir/opt/smart-screen/interface" -name '__pycache__' -type d -exec rm -rf {} + 2>/dev/null || true

	# Launcher wrapper — cd into the shared tree so relative paths work.
	install -Dm755 /dev/stdin "$pkgdir/usr/bin/turing-interface" <<'EOF'
#!/bin/bash
cd /opt/smart-screen
exec python3 /opt/smart-screen/interface/main.py "$@"
EOF

	# Desktop entry — rewrite Exec/Path to point at the wrapper.
	install -d "$pkgdir/usr/share/applications"
	sed -e 's|^Exec=.*|Exec=/usr/bin/turing-interface|' \
	    -e 's|^Path=.*|Path=/opt/smart-screen/|' \
	    scripts/turing-interface.desktop \
	    > "$pkgdir/usr/share/applications/turing-interface.desktop"
	chmod 644 "$pkgdir/usr/share/applications/turing-interface.desktop"

	# XDG hicolor icon
	install -Dm644 res/icon.svg \
		"$pkgdir/usr/share/icons/hicolor/scalable/apps/turing-screen.svg"
}
