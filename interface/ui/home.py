"""
Tela inicial — replica fiel da interface Go (home.go).

Layout:
  ┌──────────────────────────────────────────────┐
  │ HeaderBar: ☰ menu  |  "Turing Screen"        │
  ├──────────────────────────────────────────────┤
  │  ←  |   nome do tema (centro)   |  →         │
  │  [ preview image — centralizada ]            │
  │       <spacer> [Ativar] [Editar] <spacer>    │
  ├──────────────────────────────────────────────┤
  │ [Play] [Stop] [Storage]       Status ●       │
  └──────────────────────────────────────────────┘
"""
import os
import logging
import threading

from gi.repository import Gtk, Gdk, GdkPixbuf, GLib, Adw, Gio

log = logging.getLogger(__name__)


class HomePage:
    """
    Não subclassa Adw.ToolbarView (tipo final no GObject).
    Usa composição: acessa .widget para obter o Adw.ToolbarView.
    """

    def __init__(self, ws_client, themes_base: str, open_editor_cb):
        self._toolbar_view = Adw.ToolbarView()
        self._ws = ws_client
        self._themes_base = themes_base
        self._open_editor = open_editor_cb

        self._themes: list[str] = []
        self._current_index: int = 0
        self._active_theme: str = ""
        self._last_mode: str = ""
        self._connected: bool = False
        self._device_type: str = ""

        self._preview_cache: dict[str, Gdk.Texture | None] = {}

        self._build_ui()
        self._load_themes_from_disk()
        self._update_display()

        self._ws.set_event_handler(self._on_event)

    # ------------------------------------------------------------------
    # Construção da UI
    # ------------------------------------------------------------------

    def _build_ui(self):
        # ── Header bar ────────────────────────────────────────────────
        hb = Adw.HeaderBar()

        # Menu button (hambúrguer) — equivalente ao MainMenu do Go
        menu_btn = Gtk.MenuButton()
        menu_btn.set_icon_name("open-menu-symbolic")
        menu_btn.set_tooltip_text("Menu")
        menu_btn.set_popover(self._build_menu_popover())
        hb.pack_start(menu_btn)

        self._toolbar_view.add_top_bar(hb)

        # ── Corpo ─────────────────────────────────────────────────────
        body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        body.set_vexpand(True)
        body.set_margin_top(8)
        body.set_margin_bottom(0)
        body.set_margin_start(12)
        body.set_margin_end(12)

        # Nav row: ← | nome do tema | →
        nav = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        nav.set_margin_bottom(6)

        self._prev_btn = Gtk.Button()
        self._prev_btn.set_child(Gtk.Image.new_from_icon_name("go-previous-symbolic"))
        self._prev_btn.connect("clicked", lambda _: self._navigate(-1))
        nav.append(self._prev_btn)

        self._theme_label = Gtk.Label()
        self._theme_label.add_css_class("title-2")
        self._theme_label.set_hexpand(True)
        self._theme_label.set_halign(Gtk.Align.CENTER)
        self._theme_label.set_ellipsize(3)  # PANGO_ELLIPSIZE_END
        nav.append(self._theme_label)

        self._next_btn = Gtk.Button()
        self._next_btn.set_child(Gtk.Image.new_from_icon_name("go-next-symbolic"))
        self._next_btn.connect("clicked", lambda _: self._navigate(1))
        nav.append(self._next_btn)

        body.append(nav)

        # Preview image — preenche o espaço disponível
        preview_overlay = Gtk.Overlay()
        preview_overlay.set_vexpand(True)
        preview_overlay.set_hexpand(True)

        preview_bg = Gtk.Box()
        preview_bg.set_vexpand(True)
        preview_bg.set_hexpand(True)
        preview_bg.add_css_class("card")

        self._preview_pic = Gtk.Picture()
        self._preview_pic.set_hexpand(True)
        self._preview_pic.set_vexpand(True)
        self._preview_pic.set_content_fit(Gtk.ContentFit.CONTAIN)
        self._preview_pic.set_can_shrink(True)
        preview_bg.append(self._preview_pic)

        preview_overlay.set_child(preview_bg)

        self._preview_placeholder = Gtk.Label(label="Nenhum tema encontrado")
        self._preview_placeholder.add_css_class("dim-label")
        self._preview_placeholder.set_halign(Gtk.Align.CENTER)
        self._preview_placeholder.set_valign(Gtk.Align.CENTER)
        preview_overlay.add_overlay(self._preview_placeholder)

        self._preview_spinner = Gtk.Spinner()
        self._preview_spinner.set_size_request(48, 48)
        self._preview_spinner.set_halign(Gtk.Align.CENTER)
        self._preview_spinner.set_valign(Gtk.Align.CENTER)
        self._preview_spinner.set_visible(False)
        preview_overlay.add_overlay(self._preview_spinner)

        # Rótulo "ativo" no canto inferior direito do preview
        self._active_label = Gtk.Label(label="")
        self._active_label.add_css_class("osd")
        self._active_label.add_css_class("caption")
        self._active_label.set_halign(Gtk.Align.END)
        self._active_label.set_valign(Gtk.Align.END)
        self._active_label.set_margin_bottom(6)
        self._active_label.set_margin_end(8)
        preview_overlay.add_overlay(self._active_label)

        body.append(preview_overlay)

        # Botões Ativar + Editar Tema
        actions = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
        actions.set_halign(Gtk.Align.CENTER)
        actions.set_margin_top(8)
        actions.set_margin_bottom(8)

        self._activate_btn = Gtk.Button(label="Ativar")
        self._activate_btn.add_css_class("suggested-action")
        self._activate_btn.set_sensitive(False)
        self._activate_btn.connect("clicked", self._on_activate)
        actions.append(self._activate_btn)

        self._edit_btn = Gtk.Button(label="Editar Tema")
        self._edit_btn.set_sensitive(False)
        self._edit_btn.connect("clicked", self._on_open_editor)
        actions.append(self._edit_btn)

        body.append(actions)

        # ── Footer: Play | Stop | Storage ... status ──────────────────
        footer_sep = Gtk.Separator()
        body.append(footer_sep)

        footer = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=4)
        footer.set_margin_top(6)
        footer.set_margin_bottom(8)

        self._play_btn = Gtk.Button(label="Play")
        self._play_btn.set_child(self._icon_label("media-playback-start-symbolic", "Play"))
        self._play_btn.set_sensitive(False)
        self._play_btn.connect("clicked", self._on_play)
        footer.append(self._play_btn)

        self._stop_btn = Gtk.Button(label="Stop")
        self._stop_btn.set_child(self._icon_label("media-playback-stop-symbolic", "Stop"))
        self._stop_btn.set_sensitive(False)
        self._stop_btn.connect("clicked", self._on_stop)
        footer.append(self._stop_btn)

        self._storage_btn = Gtk.Button(label="Storage")
        self._storage_btn.set_child(self._icon_label("drive-harddisk-symbolic", "Storage"))
        self._storage_btn.set_sensitive(False)
        self._storage_btn.connect("clicked", self._on_storage)
        footer.append(self._storage_btn)

        spacer = Gtk.Box()
        spacer.set_hexpand(True)
        footer.append(spacer)

        self._status_label = Gtk.Label(label="Verificando...")
        self._status_label.add_css_class("caption")
        footer.append(self._status_label)

        body.append(footer)

        self._toolbar_view.set_content(body)

    @staticmethod
    def _icon_label(icon_name: str, text: str) -> Gtk.Box:
        box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=4)
        box.append(Gtk.Image.new_from_icon_name(icon_name))
        box.append(Gtk.Label(label=text))
        return box

    def _build_menu_popover(self) -> Gtk.PopoverMenu:
        """Equivalente ao buildHomeMenu() do Go."""
        menu = Gio.Menu()

        # Arquivo
        arquivo = Gio.Menu()
        s_hide = Gio.Menu()
        s_hide.append("Ocultar janela", "app.hide")
        arquivo.append_section(None, s_hide)
        s_quit = Gio.Menu()
        s_quit.append("Finalizar", "app.quit")
        s_quit.append("Finalizar e Desligar", "app.quit_off")
        arquivo.append_section(None, s_quit)
        menu.append_submenu("Arquivo", arquivo)

        # Device — espelho exato do buildHomeMenu do Go
        device = Gio.Menu()
        s_dev = Gio.Menu()
        s_dev.append("Reiniciar", "app.dev_restart")
        s_dev.append("Reboot", "app.dev_reboot")
        s_dev.append("Reset USB", "app.dev_reset")
        device.append_section(None, s_dev)
        s_off = Gio.Menu()
        s_off.append("Desligar", "app.dev_turnoff")
        device.append_section(None, s_off)
        menu.append_submenu("Device", device)

        # Configurações
        cfg = Gio.Menu()
        cfg.append("Configurações...", "app.settings")
        menu.append_submenu("Configurações", cfg)

        return Gtk.PopoverMenu.new_from_model(menu)

    # ------------------------------------------------------------------
    # Carregamento de temas
    # ------------------------------------------------------------------

    def _load_themes_from_disk(self):
        if not os.path.isdir(self._themes_base):
            log.warning("Diretório de temas não encontrado: %s", self._themes_base)
            return
        names = sorted(
            d for d in os.listdir(self._themes_base)
            if os.path.isdir(os.path.join(self._themes_base, d))
            and os.path.exists(os.path.join(self._themes_base, d, "theme.yaml"))
        )
        self._themes = names

    # ------------------------------------------------------------------
    # Navegação
    # ------------------------------------------------------------------

    def _navigate(self, direction: int):
        if not self._themes:
            return
        self._current_index = (self._current_index + direction) % len(self._themes)
        self._update_display()

    def _update_display(self):
        if not self._themes:
            self._theme_label.set_label("Nenhum tema encontrado")
            self._activate_btn.set_sensitive(False)
            self._edit_btn.set_sensitive(False)
            self._preview_placeholder.set_visible(True)
            return

        name = self._themes[self._current_index]
        self._theme_label.set_label(f"  {name}  ")
        self._edit_btn.set_sensitive(True)

        if name == self._active_theme:
            self._activate_btn.set_label("Ativo")
            self._activate_btn.set_sensitive(False)
        else:
            self._activate_btn.set_label("Ativar")
            self._activate_btn.set_sensitive(self._connected)

        self._active_label.set_label(
            f"  ▶ {self._active_theme}  " if self._active_theme else ""
        )
        self._load_preview(name)

    # ------------------------------------------------------------------
    # Preview
    # ------------------------------------------------------------------

    def _load_preview(self, name: str):
        if name in self._preview_cache:
            self._apply_preview_texture(name, self._preview_cache[name])
            return

        self._preview_placeholder.set_visible(False)
        self._preview_spinner.set_visible(True)
        self._preview_spinner.start()
        self._preview_pic.set_paintable(None)

        threading.Thread(target=self._load_preview_thread, args=(name,), daemon=True).start()

    def _load_preview_thread(self, name: str):
        theme_dir = os.path.join(self._themes_base, name)
        bg_path = self._find_background_path(theme_dir)
        texture = None
        if bg_path and os.path.exists(bg_path):
            try:
                pixbuf = GdkPixbuf.Pixbuf.new_from_file(bg_path)
                texture = Gdk.Texture.new_for_pixbuf(pixbuf)
            except Exception as e:
                log.warning("Falha ao carregar preview '%s': %s", name, e)
        self._preview_cache[name] = texture
        GLib.idle_add(self._apply_preview_texture, name, texture)

    def _find_background_path(self, theme_dir: str) -> str | None:
        # Mesma ordem exata do Go (home.go):
        #   1. assets/image_0.png  (frame extraído do vídeo)
        #   2. background.png
        for candidate in (
            os.path.join(theme_dir, "assets", "image_0.png"),
            os.path.join(theme_dir, "background.png"),
            os.path.join(theme_dir, "Background.png"),
            os.path.join(theme_dir, "BACKGROUND.png"),
        ):
            if os.path.exists(candidate):
                return candidate
        return None

    def _apply_preview_texture(self, name: str, texture: Gdk.Texture | None):
        self._preview_spinner.stop()
        self._preview_spinner.set_visible(False)

        # Ignora se o usuário já navegou para outro tema
        if not self._themes or self._themes[self._current_index] != name:
            return

        if texture:
            self._preview_placeholder.set_visible(False)
            self._preview_pic.set_paintable(texture)
        else:
            self._preview_placeholder.set_label(f"Sem preview para\n'{name}'")
            self._preview_placeholder.set_visible(True)

    # ------------------------------------------------------------------
    # WebSocket events
    # ------------------------------------------------------------------

    def _on_event(self, action: str, payload: dict):
        if action == "connection.changed":
            self._on_connection(payload.get("connected", False))
        elif action == "event.status":
            self._on_status(payload)

    def _on_connection(self, connected: bool):
        self._connected = connected
        if connected:
            self._status_label.set_label("🟢 Conectado")
        else:
            self._status_label.set_label("⚫ Desconectado")
            self._last_mode = ""
            self._set_buttons_disconnected()

    def _on_status(self, p: dict):
        mode  = p.get("mode", "")
        theme = p.get("theme", "")
        uptime = p.get("uptime", "")
        # device_type define os diretórios exibidos na tela de Storage
        dt = p.get("device_type", "")
        if dt:
            self._device_type = dt

        if mode == "editor":
            self._status_label.set_label(f"🟢 ⏸ {theme} | {uptime}")
        elif mode == "starting":
            self._status_label.set_label(f"🟢 ⏳ Aguardando device... | {uptime}")
        else:
            self._status_label.set_label(f"🟢 ▶ {theme} | {uptime}")

        if mode != self._last_mode:
            self._last_mode = mode
            if mode == "editor":
                self._set_buttons_paused()
            elif mode == "starting":
                self._set_buttons_starting()
            else:
                self._set_buttons_running()

        if theme and theme != self._active_theme:
            self._active_theme = theme
            self._update_display()

    # ------------------------------------------------------------------
    # Estado dos botões (igual ao Go)
    # ------------------------------------------------------------------

    def _set_buttons_running(self):
        self._play_btn.set_sensitive(False)
        self._stop_btn.set_sensitive(True)
        self._storage_btn.set_sensitive(True)

    def _set_buttons_paused(self):
        self._play_btn.set_sensitive(True)
        self._stop_btn.set_sensitive(False)
        self._storage_btn.set_sensitive(True)

    def _set_buttons_starting(self):
        self._play_btn.set_sensitive(False)
        self._stop_btn.set_sensitive(False)
        self._storage_btn.set_sensitive(False)

    def _set_buttons_disconnected(self):
        self._play_btn.set_sensitive(False)
        self._stop_btn.set_sensitive(False)
        self._storage_btn.set_sensitive(False)

    # ------------------------------------------------------------------
    # Ações
    # ------------------------------------------------------------------

    def _on_activate(self, _btn):
        if not self._themes:
            return
        name = self._themes[self._current_index]
        self._active_theme = name
        self._status_label.set_label(f"⏳ Aplicando: {name}...")
        self._update_display()
        self._ws.send("theme.apply", {"name": name})

    def _on_open_editor(self, _btn):
        if not self._themes:
            return
        self._open_editor(self._themes[self._current_index])

    def _on_play(self, _btn):
        self._ws.send("mode.normal", None)

    def _on_stop(self, _btn):
        self._ws.send("mode.editor", None)

    def _on_storage(self, _btn):
        from ui.device_dialog import show_device_dialog
        # Parent: a ApplicationWindow ancestral do toolbar_view
        parent = self._toolbar_view.get_root()
        show_device_dialog(self._ws, self._device_type, parent)

    # ------------------------------------------------------------------
    # Acessores públicos chamados pelo app
    # ------------------------------------------------------------------

    @property
    def widget(self) -> Adw.ToolbarView:
        return self._toolbar_view

    def notify_theme_list_dirty(self):
        """Recarrega lista de temas (chamado após salvar no editor)."""
        self._load_themes_from_disk()
        self._preview_cache.clear()
        self._update_display()
