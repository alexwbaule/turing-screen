#!/usr/bin/env python3
"""
turing-interface entry point.
Single-instance enforcement via Unix socket (same as the Go version).
"""
import logging
import os
import socket
import sys

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)

_IPC_SOCKET = "/tmp/turing-interface.sock"
_WS_PORT = 9120


def _resolve_ws_port() -> int:
    """Try to read API port from conf/config.yaml next to the binary."""
    try:
        import yaml
        cfg_path = os.path.join(os.path.dirname(__file__), "..", "conf", "config.yaml")
        cfg_path = os.path.abspath(cfg_path)
        if os.path.exists(cfg_path):
            with open(cfg_path) as f:
                cfg = yaml.safe_load(f) or {}
            return int(cfg.get("api_port", _WS_PORT))
    except Exception:
        pass
    return _WS_PORT


def main():
    # ── Single-instance: signal existing instance and exit ────────────
    try:
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.settimeout(0.3)
        sock.connect(_IPC_SOCKET)
        sock.sendall(b"show")
        sock.close()
        sys.exit(0)
    except (ConnectionRefusedError, FileNotFoundError, OSError):
        pass

    # ── First instance: claim the socket ─────────────────────────────
    try:
        os.remove(_IPC_SOCKET)
    except FileNotFoundError:
        pass

    server_sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    try:
        server_sock.bind(_IPC_SOCKET)
        server_sock.listen(1)
    except OSError as e:
        logging.warning("IPC socket bind failed: %s (continuing without single-instance)", e)
        server_sock = None

    # ── GTK imports (late, so socket is claimed first) ────────────────
    import gi
    gi.require_version("Gtk", "4.0")
    gi.require_version("Adw", "1")
    from gi.repository import GLib

    from ui.app import EditorApp

    port = _resolve_ws_port()
    app = EditorApp(ws_port=port)

    # Listen for "show" signals in a background thread
    if server_sock:
        import threading

        def _ipc_listener():
            while True:
                try:
                    conn, _ = server_sock.accept()
                    conn.recv(16)
                    conn.close()
                    GLib.idle_add(app.show_window)
                except OSError:
                    break

        t = threading.Thread(target=_ipc_listener, daemon=True)
        t.start()

    try:
        app.run()
    finally:
        if server_sock:
            server_sock.close()
        try:
            os.remove(_IPC_SOCKET)
        except FileNotFoundError:
            pass


if __name__ == "__main__":
    main()
