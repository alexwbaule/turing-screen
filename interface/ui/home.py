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

from i18n import _

log = logging.getLogger(__name__)


# Preview-thumbnail candidates, in priority order. Must match the Go home
# screen (home.go): a video frame (assets/image_0.png) wins, else the static
# background.png in any case.
_PREVIEW_CANDIDATES = (
    ("assets", "image_0.png"),
    ("background.png",),
    ("Background.png",),
    ("BACKGROUND.png",),
)


def preview_read_path(theme_dir: str) -> str | None:
    """First existing preview file in priority order, or None."""
    for parts in _PREVIEW_CANDIDATES:
        p = os.path.join(theme_dir, *parts)
        if os.path.exists(p):
            return p
    return None


def preview_write_path(theme_dir: str) -> str:
    """Where to write a regenerated preview so the loader actually picks it up.

    The loader reads the first existing candidate, so we must overwrite that
    same file (e.g. ``assets/image_0.png``) — writing ``background.png`` while
    ``image_0.png`` exists would be silently shadowed and never shown.  Falls
    back to ``background.png`` when no preview exists yet.
    """
    return preview_read_path(theme_dir) or os.path.join(theme_dir, "background.png")


class HomePage:
    """
    Não subclassa Adw.ToolbarView (tipo final no GObject).
    Usa composição: acessa .widget para obter o Adw.ToolbarView.
    """

    def __init__(self, ws_client, themes_base: str, open_editor_cb, hwinfo_cb=None):
        self._toolbar_view = Adw.ToolbarView()
        self._ws = ws_client
        self._themes_base = themes_base
        self._open_editor = open_editor_cb
        self._hwinfo_cb = hwinfo_cb  # called with dict when event.hwinfo arrives

        self._themes: list[str] = []
        self._current_index: int = 0
        self._active_theme: str = ""
        self._user_navigated: bool = False  # set once the user uses the arrows
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
        menu_btn.set_tooltip_text(_("Menu"))
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

        self._preview_placeholder = Gtk.Label(label=_("No theme found"))
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

        self._new_theme_btn = Gtk.Button(label=_("New Theme"))
        self._new_theme_btn.set_tooltip_text(_("Create a new blank theme"))
        self._new_theme_btn.connect("clicked", self._on_new_theme)
        actions.append(self._new_theme_btn)

        self._activate_btn = Gtk.Button(label=_("Activate"))
        self._activate_btn.add_css_class("suggested-action")
        self._activate_btn.set_sensitive(False)
        self._activate_btn.connect("clicked", self._on_activate)
        actions.append(self._activate_btn)

        self._edit_btn = Gtk.Button(label=_("Edit Theme"))
        self._edit_btn.set_sensitive(False)
        self._edit_btn.connect("clicked", self._on_open_editor)
        actions.append(self._edit_btn)

        self._delete_btn = Gtk.Button(label=_("Delete"))
        self._delete_btn.add_css_class("destructive-action")
        self._delete_btn.set_sensitive(False)
        self._delete_btn.set_tooltip_text(_("Delete theme from disk (cannot be undone)"))
        self._delete_btn.connect("clicked", self._on_delete)
        actions.append(self._delete_btn)

        body.append(actions)

        # ── Footer: Play | Stop | Storage ... status ──────────────────
        footer_sep = Gtk.Separator()
        body.append(footer_sep)

        footer = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=4)
        footer.set_margin_top(6)
        footer.set_margin_bottom(8)

        self._play_btn = Gtk.Button(label=_("Play"))
        self._play_btn.set_child(self._icon_label("media-playback-start-symbolic", _("Play")))
        self._play_btn.set_sensitive(False)
        self._play_btn.connect("clicked", self._on_play)
        footer.append(self._play_btn)

        self._stop_btn = Gtk.Button(label=_("Stop"))
        self._stop_btn.set_child(self._icon_label("media-playback-stop-symbolic", _("Stop")))
        self._stop_btn.set_sensitive(False)
        self._stop_btn.connect("clicked", self._on_stop)
        footer.append(self._stop_btn)

        self._storage_btn = Gtk.Button(label=_("Storage"))
        self._storage_btn.set_child(self._icon_label("drive-harddisk-symbolic", _("Storage")))
        self._storage_btn.set_sensitive(False)
        self._storage_btn.connect("clicked", self._on_storage)
        footer.append(self._storage_btn)

        spacer = Gtk.Box()
        spacer.set_hexpand(True)
        footer.append(spacer)

        self._status_label = Gtk.Label(label=_("Checking..."))
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

        # File
        arquivo = Gio.Menu()
        s_hide = Gio.Menu()
        s_hide.append(_("Hide window"), "app.hide")
        arquivo.append_section(None, s_hide)
        s_quit = Gio.Menu()
        s_quit.append(_("Quit"), "app.quit")
        s_quit.append(_("Quit and Turn Off"), "app.quit_off")
        arquivo.append_section(None, s_quit)
        menu.append_submenu(_("File"), arquivo)

        # Device
        device = Gio.Menu()
        s_dev = Gio.Menu()
        s_dev.append(_("Restart"), "app.dev_restart")
        s_dev.append(_("Reboot"), "app.dev_reboot")
        s_dev.append(_("Reset USB"), "app.dev_reset")
        device.append_section(None, s_dev)
        s_off = Gio.Menu()
        s_off.append(_("Turn Off"), "app.dev_turnoff")
        device.append_section(None, s_off)
        menu.append_submenu(_("Device"), device)

        # Settings
        cfg = Gio.Menu()
        cfg.append(_("Settings..."), "app.settings")
        menu.append_submenu(_("Settings"), cfg)

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
        self._user_navigated = True
        self._current_index = (self._current_index + direction) % len(self._themes)
        self._update_display()

    def _update_display(self):
        if not self._themes:
            self._theme_label.set_label(_("No theme found"))
            self._activate_btn.set_sensitive(False)
            self._edit_btn.set_sensitive(False)
            self._delete_btn.set_sensitive(False)
            self._preview_placeholder.set_visible(True)
            return

        name = self._themes[self._current_index]
        self._theme_label.set_label(f"  {name}  ")
        self._edit_btn.set_sensitive(True)

        # Can't delete the active theme — the daemon references it and the
        # config points at it; deleting would leave a dangling reference.
        is_active = name == self._active_theme
        self._delete_btn.set_sensitive(not is_active)
        self._delete_btn.set_tooltip_text(
            _("Activate another theme before deleting this one.") if is_active
            else _("Delete theme from disk (cannot be undone)")
        )

        if name == self._active_theme:
            self._activate_btn.set_label(_("Active"))
            self._activate_btn.set_sensitive(False)
        else:
            self._activate_btn.set_label(_("Activate"))
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
        return preview_read_path(theme_dir)

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
            self._preview_placeholder.set_label(f"{_('No preview for')}\n'{name}'")
            self._preview_placeholder.set_visible(True)

    # ------------------------------------------------------------------
    # WebSocket events
    # ------------------------------------------------------------------

    def _on_event(self, action: str, payload: dict):
        if action == "connection.changed":
            self._on_connection(payload.get("connected", False))
        elif action == "event.status":
            self._on_status(payload)
        elif action == "event.hwinfo":
            if self._hwinfo_cb:
                self._hwinfo_cb(payload)

    def _on_connection(self, connected: bool):
        self._connected = connected
        if connected:
            self._status_label.set_label(f"🟢 {_('Connected')}")
        else:
            self._status_label.set_label(f"⚫ {_('Disconnected')}")
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
            self._status_label.set_label(f"🟢 ⏳ {_('Waiting for device...')} | {uptime}")
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
            # Start on the daemon's active theme instead of the first in the
            # list — but only until the user navigates manually.
            if not self._user_navigated and theme in self._themes:
                self._current_index = self._themes.index(theme)
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
        self._status_label.set_label(f"⏳ {_('Applying')}: {name}...")
        self._update_display()
        self._ws.send("theme.apply", {"name": name})

    def _on_open_editor(self, _btn):
        if not self._themes:
            return
        self._open_editor(self._themes[self._current_index])

    def _on_new_theme(self, _btn):
        dlg = Adw.AlertDialog()
        dlg.set_heading(_("New theme"))
        dlg.set_body(_("New theme name:"))
        entry = Gtk.Entry()
        entry.set_placeholder_text(_("e.g. MyTheme"))
        try:
            dlg.set_extra_child(entry)
        except AttributeError:
            # Older libadwaita without extra_child — fall back to body-only.
            pass
        dlg.add_response("cancel", _("Cancel"))
        dlg.add_response("create", _("Create"))
        dlg.set_response_appearance("create", Adw.ResponseAppearance.SUGGESTED)
        try:
            dlg.set_default_response("create")
        except AttributeError:
            pass

        def on_resp(d, r):
            if r == "create":
                self._do_create_theme(entry.get_text().strip())
        dlg.connect("response", on_resp)
        parent = self._toolbar_view.get_root()
        if parent:
            dlg.present(parent)

    def _do_create_theme(self, name: str):
        import re
        from theme.models import Theme, Display
        # Sanitize: letters, digits, spaces, dash, underscore only.
        clean = re.sub(r"[^\w\- ]", "", name).strip()
        if not clean:
            self._show_message(_("Invalid name"),
                               _("Enter a valid name for the theme."))
            return
        theme_dir = os.path.join(self._themes_base, clean)
        if os.path.exists(theme_dir):
            self._show_message(_("Already exists"),
                               f"'{clean}'")
            return
        try:
            os.makedirs(theme_dir, exist_ok=True)
            theme = Theme(display=Display(SIZE='5"', ORIENTATION="landscape",
                                          WIDTH=1280, HEIGHT=720))
            theme.save(os.path.join(theme_dir, "theme.yaml"))
        except Exception as e:
            self._show_message(_("Failed to create"),
                               f"{e}\n\nVerifique as permissões em {self._themes_base}.")
            return
        log.info("Novo tema criado: %s", theme_dir)
        self._load_themes_from_disk()
        self._preview_cache.clear()
        if clean in self._themes:
            self._current_index = self._themes.index(clean)
        self._update_display()
        # Jump straight into the editor for the new blank theme.
        self._open_editor(clean)

    def _show_message(self, heading: str, body: str):
        dlg = Adw.AlertDialog()
        dlg.set_heading(heading)
        dlg.set_body(body)
        dlg.add_response("ok", "OK")
        parent = self._toolbar_view.get_root()
        if parent:
            dlg.present(parent)

    def _on_delete(self, _btn):
        if not self._themes:
            return
        name = self._themes[self._current_index]
        if name == self._active_theme:
            return  # button is disabled in this case; defensive guard

        dlg = Adw.AlertDialog()
        dlg.set_heading(_("Delete theme"))
        dlg.set_body(
            f"'{name}'\nres/themes/{name}"
        )
        dlg.add_response("cancel", _("Cancel"))
        dlg.add_response("delete", _("Delete"))
        dlg.set_response_appearance("delete", Adw.ResponseAppearance.DESTRUCTIVE)
        dlg.connect("response",
                    lambda d, r: self._do_delete(name) if r == "delete" else None)
        parent = self._toolbar_view.get_root()
        if parent:
            dlg.present(parent)

    def _do_delete(self, name: str):
        import shutil
        theme_dir = os.path.join(self._themes_base, name)
        try:
            shutil.rmtree(theme_dir)
        except Exception as e:
            err = Adw.AlertDialog()
            err.set_heading(_("Failed to delete"))
            err.set_body(f"{e}\n\n{self._themes_base}")
            err.add_response("ok", "OK")
            parent = self._toolbar_view.get_root()
            if parent:
                err.present(parent)
            return
        log.info("Tema excluído: %s", theme_dir)
        # Clamp index (the deleted theme may have been last) then reload.
        self._load_themes_from_disk()
        if self._current_index >= len(self._themes):
            self._current_index = max(0, len(self._themes) - 1)
        self._preview_cache.clear()
        self._update_display()

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
