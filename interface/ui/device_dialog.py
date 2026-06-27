"""
Tela de Storage do Device — espelho de device_dialog.go (Go/Fyne).

Lista storage (total/usado/livre), arquivos remotos, upload/download,
play/stop de vídeo e reinício do device — tudo via WebSocket.
"""
import base64
import logging

from gi.repository import Gtk, Adw, Gio, GLib, GObject

log = logging.getLogger(__name__)


def _format_bytes(b: int) -> str:
    if b is None:
        return "--"
    b = int(b)
    if b < 1024:
        return f"{b} B"
    if b < 1024 * 1024:
        return f"{b / 1024:.1f} KB"
    if b < 1024 * 1024 * 1024:
        return f"{b / (1024 * 1024):.1f} MB"
    return f"{b / (1024 * 1024 * 1024):.1f} GB"


class DeviceDialog(Gtk.Window):
    """Janela modal de gerenciamento do storage do device."""

    def __init__(self, ws_client, device_type: str, parent):
        super().__init__(transient_for=parent, modal=False, destroy_with_parent=True)
        self._ws = ws_client
        self._device_type = device_type or ""
        self.set_title("Device Storage")
        self.set_default_size(700, 440)

        if self._device_type == "turzx":
            self._current_dir = "/tmp/sdcard/mmcblk0p1/video/"
            self._dirs = [
                "/tmp/sdcard/mmcblk0p1/video/",
                "/tmp/sdcard/mmcblk0p1/img/",
            ]
        else:
            self._current_dir = "/root/video/"
            self._dirs = ["/root/video/", "/root/image/", "/root/font/", "/root/"]

        self._files: list[str] = []
        self._build_ui()

        # Carrega storage ao abrir
        GLib.idle_add(self._refresh_storage)

    # ------------------------------------------------------------------
    # UI
    # ------------------------------------------------------------------

    def _build_ui(self):
        header = Gtk.HeaderBar()
        self.set_titlebar(header)

        root = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=0)

        # ── Painel esquerdo: Storage + Reiniciar ──
        left = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        left.set_margin_start(10)
        left.set_margin_end(10)
        left.set_margin_top(10)
        left.set_margin_bottom(10)
        left.set_size_request(190, -1)

        lbl = Gtk.Label(label="Storage")
        lbl.set_halign(Gtk.Align.START)
        lbl.add_css_class("heading")
        left.append(lbl)

        self._total_lbl = Gtk.Label(label="Total:  --")
        self._used_lbl = Gtk.Label(label="Usado: --")
        self._free_lbl = Gtk.Label(label="Livre:  --")
        for l in (self._total_lbl, self._used_lbl, self._free_lbl):
            l.set_halign(Gtk.Align.START)
            left.append(l)

        btn_storage = Gtk.Button(label="Refresh Storage")
        btn_storage.connect("clicked", lambda *_: self._refresh_storage())
        left.append(btn_storage)

        left.append(Gtk.Separator(orientation=Gtk.Orientation.HORIZONTAL))

        btn_restart = Gtk.Button(label="Reiniciar Device")
        btn_restart.connect("clicked", lambda *_: self._restart_device())
        left.append(btn_restart)

        root.append(left)
        root.append(Gtk.Separator(orientation=Gtk.Orientation.VERTICAL))

        # ── Painel direito: diretório + lista + ações ──
        right = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        right.set_margin_start(10)
        right.set_margin_end(10)
        right.set_margin_top(10)
        right.set_margin_bottom(10)
        right.set_hexpand(True)
        right.set_vexpand(True)

        self._dir_combo = Gtk.DropDown.new_from_strings(self._dirs)
        self._dir_combo.set_selected(self._dirs.index(self._current_dir) if self._current_dir in self._dirs else 0)
        self._dir_combo.connect("notify::selected", self._on_dir_changed)
        right.append(self._dir_combo)

        # Lista de arquivos
        self._files_model = Gio.ListStore.new(_FileItem)
        selection = Gtk.SingleSelection.new(self._files_model)
        factory = Gtk.SignalListItemFactory()
        factory.connect("setup", self._on_factory_setup)
        factory.connect("bind", self._on_factory_bind)
        self._list = Gtk.ListView.new(selection, factory)
        self._list.set_vexpand(True)
        self._list.connect("activate", self._on_row_activated)

        list_scroll = Gtk.ScrolledWindow()
        list_scroll.set_child(self._list)
        list_scroll.set_min_content_height(160)
        right.append(list_scroll)

        # Ações de arquivo
        actions = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        actions.set_margin_top(6)
        for label, cb in (
            ("Refresh Arquivos", lambda *_: self._refresh_files()),
            ("Upload",           lambda *_: self._upload_file()),
            ("Deletar",          lambda *_: self._delete_file()),
            ("Play Video",       lambda *_: self._play_selected()),
            ("Stop Video",       lambda *_: self._stop_playback()),
        ):
            b = Gtk.Button(label=label)
            b.connect("clicked", cb)
            actions.append(b)
        right.append(actions)

        root.append(right)

        # ── Status label embaixo (largura total) ──
        outer = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        outer.append(root)
        outer.append(Gtk.Separator())
        self._status = Gtk.Label(label="Verificando...")
        self._status.set_halign(Gtk.Align.START)
        self._status.set_margin_start(10)
        self._status.set_margin_end(10)
        self._status.set_margin_top(6)
        self._status.set_margin_bottom(8)
        self._status.set_ellipsize(3)  # PANGO_ELLIPSIZE_END
        outer.append(self._status)

        self.set_child(outer)

    # ListView factory -------------------------------------------------

    def _on_factory_setup(self, _factory, item):
        item.set_child(Gtk.Label(xalign=0.0))

    def _on_factory_bind(self, _factory, item):
        label = item.get_child()
        label.set_label(item.get_item().name)

    def _on_row_activated(self, _list, pos):
        # seleção via click — SingleSelection já cuida disso
        pass

    # ------------------------------------------------------------------
    # Diretório
    # ------------------------------------------------------------------

    def _on_dir_changed(self, combo, _pspec):
        idx = combo.get_selected()
        if 0 <= idx < len(self._dirs):
            self._current_dir = self._dirs[idx]
            self._refresh_files()

    # ------------------------------------------------------------------
    # WS: storage.info
    # ------------------------------------------------------------------

    def _refresh_storage(self):
        if not self._ws or not self._ws.is_connected():
            self._set_status("⚫ Não conectado")
            self._total_lbl.set_label("Total:  --")
            self._used_lbl.set_label("Usado: --")
            self._free_lbl.set_label("Livre:  --")
            return
        self._set_status("⏳ Lendo storage...")
        self._ws.send("storage.info", None, callback=self._on_storage_info)

    def _on_storage_info(self, payload, error):
        if error:
            self._set_status(f"Erro: {error}")
            return
        info = payload or {}
        self._total_lbl.set_label(f"Total:  {_format_bytes(info.get('total'))}")
        self._used_lbl.set_label(f"Usado: {_format_bytes(info.get('used'))}")
        self._free_lbl.set_label(f"Livre:  {_format_bytes(info.get('free'))}")
        self._set_status("🟢 Conectado")

    # ------------------------------------------------------------------
    # WS: storage.files
    # ------------------------------------------------------------------

    def _refresh_files(self):
        if not self._ws or not self._ws.is_connected():
            return
        self._ws.send("storage.files", {"path": self._current_dir},
                      callback=self._on_files)

    def _on_files(self, payload, error):
        if error:
            self._set_status(f"Erro: {error}")
            return
        files = (payload or {}).get("files", []) or []
        self._files = list(files)
        self._files_model.remove_all()
        for f in self._files:
            self._files_model.append(_FileItem(f))
        self._set_status(f"{len(self._files)} arquivo(s)")

    # ------------------------------------------------------------------
    # WS: storage.upload
    # ------------------------------------------------------------------

    def _upload_file(self):
        dialog = Gtk.FileDialog()
        dialog.set_title("Enviar para o device")
        f = Gtk.FileFilter()
        f.set_name("Vídeos")
        for ext in ("*.mp4", "*.avi", "*.mkv", "*.png"):
            f.add_pattern(ext)
        store = Gio.ListStore.new(Gtk.FileFilter)
        store.append(f)
        dialog.set_filters(store)
        dialog.open(self, None, self._on_upload_chosen)

    def _on_upload_chosen(self, dialog, result):
        try:
            file = dialog.open_finish(result)
        except Exception:
            return
        if not file:
            return
        path = file.get_path()
        name = file.get_basename()
        try:
            with open(path, "rb") as fh:
                data = fh.read()
        except Exception as e:
            self._set_status(f"Erro ao ler: {e}")
            return
        self._set_status(f"Enviando {name}...")
        b64 = base64.b64encode(data).decode("ascii")
        self._ws.send("storage.upload", {"name": name, "data": b64},
                      callback=lambda p, err: self._on_uploaded(name, p, err))

    def _on_uploaded(self, name, _payload, error):
        if error:
            self._set_status(f"Erro: {error}")
        else:
            self._set_status(f"✓ {name} enviado")
            self._refresh_files()

    # ------------------------------------------------------------------
    # WS: storage.delete
    # ------------------------------------------------------------------

    def _selected_file_name(self) -> str | None:
        sel = self._list.get_model()
        if sel is None:
            return None
        item = sel.get_selected_item()
        if item is None:
            return None
        return item.name

    def _delete_file(self):
        name = self._selected_file_name()
        if not name:
            return
        full = self._current_dir + name
        self._ws.send("storage.delete", {"path": full},
                      callback=lambda p, err: self._on_deleted(name, p, err))

    def _on_deleted(self, name, _payload, error):
        if error:
            self._set_status(f"Erro: {error}")
        else:
            self._set_status(f"✓ Deletado: {name}")
            self._refresh_files()

    # ------------------------------------------------------------------
    # WS: theme.video.start / stop
    # ------------------------------------------------------------------

    def _play_selected(self):
        name = self._selected_file_name()
        if not name:
            return
        full = self._current_dir + name
        self._ws.send("theme.video.start", {"path": full},
                      callback=lambda p, err: self._on_played(name, err))

    def _on_played(self, name, error):
        if error:
            self._set_status(f"Erro: {error}")
        else:
            self._set_status(f"▶ Playing: {name}")

    def _stop_playback(self):
        self._ws.send("theme.video.stop", None,
                      callback=lambda p, err: self._set_status("⏹ Vídeo parado") if not err
                      else self._set_status(f"Erro: {err}"))

    # ------------------------------------------------------------------
    # WS: device.restart
    # ------------------------------------------------------------------

    def _restart_device(self):
        if not self._ws or not self._ws.is_connected():
            return
        self._ws.send("device.restart", None,
                      callback=lambda p, err: self._set_status("Device reiniciado") if not err
                      else self._set_status(f"Erro: {err}"))

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _set_status(self, text: str):
        self._status.set_label(text)


class _FileItem(GObject.Object):
    """Wrapper para Gtk.ListView via Gio.ListStore."""
    __gtype_name__ = "DeviceFileItem"

    def __init__(self, name: str):
        super().__init__()
        self.name = name


def show_device_dialog(ws_client, device_type: str, parent):
    """Cria e mostra a janela de storage do device."""
    DeviceDialog(ws_client, device_type, parent).present()
