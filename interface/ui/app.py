"""
Main EditorApp — janela principal (home) e janela do editor.
"""
import os
import logging

from gi.repository import Gtk, Adw, GLib, Gio, Gdk

from ws_client import WSClient
from ui.home import HomePage
from ui.canvas import ThemeCanvas
from ui.properties import PropertiesPanel
from ui.layers_panel import LayersPanel
from ui.toolbox import Toolbox
from ui.draggable import load_font_cache
from theme.models import Theme

log = logging.getLogger(__name__)

APP_ID = "me.abaule.turing-screen"


def _register_fonts(fonts_dir: str):
    """
    Registra fontes TTF/OTF via fontconfig (fc-cache) e força o Pango a
    recarregar o mapa de fontes. GTK4 não suporta @font-face via CSS.
    """
    if not os.path.isdir(fonts_dir):
        return
    import subprocess
    # Escreve config de usuário apontando para nossa pasta de fontes
    conf_dir = os.path.expanduser("~/.config/fontconfig/conf.d")
    os.makedirs(conf_dir, exist_ok=True)
    conf_file = os.path.join(conf_dir, "99-turing-screen.conf")
    abs_dir = os.path.abspath(fonts_dir)
    with open(conf_file, "w") as f:
        f.write(f"""<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <dir>{abs_dir}</dir>
</fontconfig>
""")
    result = subprocess.run(
        ["fc-cache", "-f", abs_dir], capture_output=True, timeout=15
    )
    if result.returncode == 0:
        # Força o Pango a recarregar o mapa de fontes no processo atual
        try:
            from gi.repository import PangoCairo
            font_map = PangoCairo.font_map_get_default()
            font_map.config_changed()
        except Exception:
            pass
        log.info("Fontes registradas via fontconfig: %s", abs_dir)
    else:
        log.warning("fc-cache falhou: %s", result.stderr.decode())


def _find_themes_base() -> str:
    """
    Resolve o diretório base dos temas: res/themes/ relativo ao CWD
    (o daemon roda a partir da raiz do projeto, assim como o Go).
    """
    candidates = [
        os.path.join(os.getcwd(), "res", "themes"),
        os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), "res", "themes"),
        "/usr/share/turing-screen/themes",
    ]
    for p in candidates:
        if os.path.isdir(p):
            return p
    # Fallback: retorna o primeiro mesmo não existindo, para dar mensagem clara
    return candidates[0]


class EditorApp:

    def __init__(self, ws_port: int = 9120):
        self._ws = WSClient(port=ws_port)
        self._app = Adw.Application(application_id=APP_ID)
        self._app.connect("activate", self._on_activate)

        self._themes_base = _find_themes_base()
        log.info("Themes base: %s", self._themes_base)

        self._main_window: Adw.ApplicationWindow | None = None
        self._editor_window: Gtk.Window | None = None
        self._home: HomePage | None = None

        self._canvas: ThemeCanvas | None = None
        self._props: PropertiesPanel | None = None
        self._layers: LayersPanel | None = None

        self._current_theme: Theme | None = None
        self._theme_dir: str = ""
        self._theme_name: str = ""

        self._tray_icon = None   # TrayIcon D-Bus (se disponível)

    # ------------------------------------------------------------------
    # Run / show
    # ------------------------------------------------------------------

    def run(self):
        self._ws.start()
        self._app.run(None)

    def show_window(self):
        if self._main_window:
            self._main_window.present()

    def _on_main_close(self, _window) -> bool:
        """Intercepta o fechamento da janela principal → oculta em vez de fechar."""
        self._main_window.hide()
        return True  # True = bloqueia o destroy

    # ------------------------------------------------------------------
    # System tray
    # ------------------------------------------------------------------

    def _start_tray(self):
        try:
            from ui.tray import TrayIcon
        except Exception as e:
            log.warning("Tray: import falhou: %s", e)
            return

        def _toggle():
            if self._main_window:
                if self._main_window.get_visible():
                    self._main_window.hide()
                else:
                    self._main_window.present()

        def _quit():
            self._app.quit()

        def _quit_off():
            self._ws.send("device.turnoff", None)
            self._app.quit()

        self._tray_icon = TrayIcon(
            icon_name="turing-screen",
            title="Turing Screen",
            on_activate=_toggle,
            on_quit=_quit,
            on_quit_off=_quit_off,
            on_context_menu=self._show_tray_menu,
        )
        self._tray_icon.start()
        log.info("System tray iniciado via D-Bus")

    def _show_tray_menu(self, x: int, y: int):
        """Mostra menu de contexto do tray como Gtk.PopoverMenu."""
        if not self._main_window:
            return

        menu_model = Gio.Menu()

        s1 = Gio.Menu()
        s1.append("Mostrar / Ocultar", "app.hide")
        menu_model.append_section(None, s1)

        s2 = Gio.Menu()
        s2.append("Finalizar", "app.quit")
        s2.append("Finalizar e Desligar", "app.quit_off")
        menu_model.append_section(None, s2)

        # Ancora o popover no conteúdo da janela principal
        anchor = self._main_window.get_content()
        if anchor is None:
            anchor = self._main_window
        popover = Gtk.PopoverMenu.new_from_model(menu_model)
        popover.set_has_arrow(False)
        popover.set_parent(anchor)
        self._main_window.present()
        popover.popup()

    # ------------------------------------------------------------------
    # Main window
    # ------------------------------------------------------------------

    def _on_activate(self, app):
        # Registra fontes customizadas antes de criar widgets
        fonts_dir = os.path.join(
            os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))),
            "res", "fonts",
        )
        _register_fonts(fonts_dir)
        # Build the font-path → (family, weight) cache from fontconfig so the
        # preview renders the real font (e.g. "Helvetica Neue LT Pro" Black),
        # not the default fallback.
        load_font_cache(fonts_dir)

        self._main_window = Adw.ApplicationWindow(application=app)
        self._main_window.set_title("Turing Screen")
        self._main_window.set_default_size(520, 480)
        self._main_window.set_resizable(True)

        # Fechar janela → ocultar no tray (não sair), igual ao Go
        self._main_window.connect("close-request", self._on_main_close)

        self._home = HomePage(
            ws_client=self._ws,
            themes_base=self._themes_base,
            open_editor_cb=self._open_editor_for_theme,
        )
        self._main_window.set_content(self._home.widget)

        # GActions referenciadas pelo menu do HomePage
        self._register_actions(app)

        # System tray (pystray, opcional)
        self._start_tray()

        self._main_window.present()

    # ------------------------------------------------------------------
    # Editor window
    # ------------------------------------------------------------------

    def _open_editor_for_theme(self, theme_name: str):
        theme_dir = os.path.join(self._themes_base, theme_name)
        yaml_path = os.path.join(theme_dir, "theme.yaml")

        if not os.path.isdir(theme_dir):
            self._show_error(f"Diretório do tema não encontrado:\n{theme_dir}")
            return
        if not os.path.exists(yaml_path):
            self._show_error(f"theme.yaml não encontrado em:\n{theme_dir}")
            return

        try:
            theme = Theme.load(yaml_path)
        except Exception as e:
            self._show_error(f"Falha ao carregar tema:\n{e}")
            return

        self._current_theme = theme
        self._theme_dir = theme_dir
        self._theme_name = theme_name

        # Calcula tamanho da janela como o Go faz: canvas_w + 570, canvas_h + 60
        # (180px toolbox + 350px props + separators/padding)
        canvas_w = (theme.display.WIDTH  if theme.display and theme.display.WIDTH  else 1280)
        canvas_h = (theme.display.HEIGHT if theme.display and theme.display.HEIGHT else 720)
        win_w = canvas_w + 570
        win_h = canvas_h + 60

        if self._editor_window is None:
            self._build_editor_window()

        self._editor_window.set_default_size(win_w, win_h)
        self._canvas.load_theme(theme, theme_dir)
        self._editor_window.set_title(f"Editor — {theme_name}")
        self._editor_window.present()

    def _build_editor_window(self):
        self._editor_window = Gtk.ApplicationWindow(application=self._app)
        self._editor_window.set_title("Theme Editor")
        self._editor_window.connect("close-request", self._on_editor_close)
        self._register_editor_actions()

        # Painel direito (Properties + Layers) — criado antes do canvas
        self._props = PropertiesPanel()
        self._layers = LayersPanel(canvas=None)

        # Canvas (centro)
        self._canvas = ThemeCanvas(self._props, self._layers)
        self._layers._canvas = self._canvas

        # Wire delete button in properties panel → canvas.remove_element
        if hasattr(self._props, "set_on_delete"):
            self._props.set_on_delete(self._canvas.remove_element)
        if hasattr(self._props, "set_on_type_change"):
            self._props.set_on_type_change(self._canvas.swap_element_type)

        # Toolbox (esquerda)
        self._toolbox = Toolbox(add_element_cb=self._canvas.add_new_element)

        # ── Layout: Toolbox | Canvas | (Properties acima / Layers abaixo) ──
        root = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=0)

        root.append(self._toolbox)
        root.append(Gtk.Separator(orientation=Gtk.Orientation.VERTICAL))
        root.append(self._canvas)   # hexpand=True → ocupa o centro
        root.append(Gtk.Separator(orientation=Gtk.Orientation.VERTICAL))

        # Painel direito: largura fixa 350px (≈ Go 320px scroll + padding)
        right = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        right.set_size_request(350, -1)
        right.set_hexpand(False)

        self._props.set_vexpand(True)
        self._props.set_hexpand(False)
        right.append(self._props)
        right.append(Gtk.Separator())
        self._layers.set_size_request(350, 220)
        self._layers.set_hexpand(False)
        right.append(self._layers)
        root.append(right)

        # ── Header bar ──
        header = Gtk.HeaderBar()
        header.set_show_title_buttons(True)

        save_btn = Gtk.Button(label="Salvar")
        save_btn.add_css_class("suggested-action")
        save_btn.connect("clicked", self._on_save)
        header.pack_end(save_btn)

        # Menus Arquivo e Editar (equivalente ao buildMainMenu do Go)
        arquivo_btn = Gtk.MenuButton(label="Arquivo")
        arquivo_btn.set_popover(self._build_editor_arquivo_menu())
        header.pack_start(arquivo_btn)

        editar_btn = Gtk.MenuButton(label="Editar")
        editar_btn.set_popover(self._build_editor_editar_menu())
        header.pack_start(editar_btn)

        self._editor_window.set_titlebar(header)
        self._editor_window.set_child(root)

    def _on_editor_close(self, _window):
        self._editor_window.hide()
        if self._home:
            self._home.notify_theme_list_dirty()
        return True  # bloqueia destroy — re-abrir é mais rápido

    # ------------------------------------------------------------------
    # Editor window menus (Arquivo / Editar) — equivalente ao buildMainMenu Go
    # ------------------------------------------------------------------

    def _register_editor_actions(self):
        """Registra GActions no nível da janela do editor (prefixo 'win.')."""
        win = self._editor_window
        def _waction(name, cb):
            a = Gio.SimpleAction.new(name, None)
            a.connect("activate", lambda *_: cb())
            win.add_action(a)

        _waction("new_theme",    self._editor_new_theme)
        _waction("open_theme",   self._editor_open_theme)
        _waction("save",         lambda: self._on_save(None))
        _waction("save_as",      self._editor_save_as)
        _waction("close_theme",  self._editor_close_theme)
        _waction("close_editor", lambda: self._on_editor_close(None))
        _waction("bg_image",     self._editor_add_bg_image)
        _waction("bg_video",     self._editor_add_bg_video)
        _waction("delete_comp",  self.delete_selected_widget)

    def _build_editor_arquivo_menu(self) -> Gtk.PopoverMenu:
        menu = Gio.Menu()
        s1 = Gio.Menu()
        s1.append("Novo Tema", "win.new_theme")
        s1.append("Abrir Tema...", "win.open_theme")
        menu.append_section(None, s1)
        s2 = Gio.Menu()
        s2.append("Salvar", "win.save")
        s2.append("Salvar Como...", "win.save_as")
        menu.append_section(None, s2)
        s3 = Gio.Menu()
        s3.append("Fechar Tema", "win.close_theme")
        s3.append("Fechar Editor", "win.close_editor")
        menu.append_section(None, s3)
        return Gtk.PopoverMenu.new_from_model(menu)

    def _build_editor_editar_menu(self) -> Gtk.PopoverMenu:
        menu = Gio.Menu()
        bg = Gio.Menu()
        bg.append("Adicionar Imagem", "win.bg_image")
        bg.append("Adicionar Vídeo", "win.bg_video")
        menu.append_submenu("Background", bg)
        sep = Gio.Menu()
        sep.append("Excluir Componente", "win.delete_comp")
        menu.append_section(None, sep)
        return Gtk.PopoverMenu.new_from_model(menu)

    def _editor_new_theme(self):
        from theme.models import Theme as ThemeModel
        self._current_theme = ThemeModel()
        self._theme_dir = ""
        self._theme_name = ""
        if self._canvas:
            self._canvas.load_theme(self._current_theme, "")
        if self._editor_window:
            self._editor_window.set_title("Editor — Novo Tema")

    def _editor_open_theme(self):
        dialog = Gtk.FileDialog()
        dialog.set_title("Abrir Tema")
        f = Gtk.FileFilter()
        f.set_name("YAML (*.yaml, *.yml)")
        f.add_pattern("*.yaml")
        f.add_pattern("*.yml")
        store = Gio.ListStore.new(Gtk.FileFilter)
        store.append(f)
        dialog.set_filters(store)
        dialog.open(self._editor_window, None, self._on_open_theme_done)

    def _on_open_theme_done(self, dialog, result):
        try:
            file = dialog.open_finish(result)
        except Exception:
            return
        if not file:
            return
        yaml_path = file.get_path()
        try:
            theme = Theme.load(yaml_path)
        except Exception as e:
            self._show_error(f"Falha ao abrir tema:\n{e}")
            return
        self._current_theme = theme
        self._theme_dir = os.path.dirname(yaml_path)
        self._theme_name = os.path.basename(self._theme_dir)
        if self._canvas:
            self._canvas.load_theme(theme, self._theme_dir)
        if self._editor_window:
            self._editor_window.set_title(f"Editor — {self._theme_name}")

    def _editor_save_as(self):
        dialog = Gtk.FileDialog()
        dialog.set_title("Salvar Como")
        dialog.set_initial_name("theme.yaml")
        dialog.save(self._editor_window, None, self._on_save_as_done)

    def _on_save_as_done(self, dialog, result):
        try:
            file = dialog.save_finish(result)
        except Exception:
            return
        if not file:
            return
        path = file.get_path()
        self._theme_dir = os.path.dirname(path)
        self._theme_name = os.path.basename(self._theme_dir)
        if not self._current_theme:
            return
        self.capture_z_order()
        try:
            self._current_theme.save(path)
            self._toast(f"Salvo em {os.path.basename(path)}")
        except Exception as e:
            self._show_error(f"Falha ao salvar:\n{e}")

    def _editor_close_theme(self):
        self._editor_new_theme()

    def _editor_add_bg_image(self):
        dialog = Gtk.FileDialog()
        dialog.set_title("Selecionar Imagem de Fundo")
        f = Gtk.FileFilter()
        f.set_name("PNG")
        f.add_pattern("*.png")
        store = Gio.ListStore.new(Gtk.FileFilter)
        store.append(f)
        dialog.set_filters(store)
        dialog.open(self._editor_window, None, self._on_bg_image_done)

    def _on_bg_image_done(self, dialog, result):
        try:
            file = dialog.open_finish(result)
        except Exception:
            return
        if file and self._canvas:
            self._canvas.add_background_layer(file.get_path())

    def _editor_add_bg_video(self):
        self._toast("Vídeo de fundo ainda não suportado nesta versão.")

    # ------------------------------------------------------------------
    # GActions (menu do HomePage)
    # ------------------------------------------------------------------

    def _register_actions(self, app):
        def _action(name, cb):
            a = Gio.SimpleAction.new(name, None)
            a.connect("activate", cb)
            app.add_action(a)

        _action("hide",           lambda *_: self._on_main_close(self._main_window))
        _action("quit",           lambda *_: app.quit())
        _action("quit_off",       self._on_quit_off)
        _action("dev_restart",    lambda *_: self._ws_cmd("device.restart",  "Reiniciando..."))
        _action("dev_reboot",     lambda *_: self._ws_cmd("device.reboot",   "Rebooting..."))
        _action("dev_reset",      lambda *_: self._ws_cmd("device.reset",    "Resetando USB..."))
        _action("dev_turnoff",    lambda *_: self._ws_cmd("device.turnoff",  "Desligando..."))
        _action("settings",       lambda *_: self._show_config())
        _action("delete_widget",  lambda *_: self.delete_selected_widget())

    def _on_quit_off(self, *_):
        self._ws.send("device.turnoff", None)
        self._app.quit()

    def _show_config(self):
        from ui.config_dialog import show_config_dialog
        show_config_dialog(self._main_window)

    def _ws_cmd(self, action: str, pending: str):
        if self._home:
            self._home._status_label.set_label(f"⏳ {pending}")
        self._ws.send(action, None, callback=self._on_ws_cmd_done)

    def _on_ws_cmd_done(self, payload, error):
        if self._home:
            msg = f"⚠ Erro: {error}" if error else "✓ Concluído"
            self._home._status_label.set_label(msg)

    # ------------------------------------------------------------------
    # Canvas helpers
    # ------------------------------------------------------------------

    def select_widget_by_path(self, path: str):
        """Select a canvas element by its YAML path."""
        if self._canvas:
            self._canvas.select_widget_by_path(path)

    def capture_z_order(self):
        """Write current stack position as INDEX on each element's data."""
        if self._canvas:
            self._canvas.capture_z_order()

    def delete_selected_widget(self):
        """Delete the currently selected canvas element (with confirm dialog)."""
        if not self._canvas or not self._canvas._selected:
            return
        elem = self._canvas._selected
        dialog = Adw.AlertDialog()
        dialog.set_heading("Excluir Componente")
        dialog.set_body(f"Excluir '{elem.yaml_path}'?")
        dialog.add_response("cancel", "Cancelar")
        dialog.add_response("delete", "Excluir")
        dialog.set_response_appearance("delete", Adw.ResponseAppearance.DESTRUCTIVE)
        dialog.connect(
            "response",
            lambda d, r: self._canvas.remove_element(elem) if r == "delete" else None
        )
        parent = self._editor_window or self._main_window
        if parent:
            dialog.present(parent)

    # ------------------------------------------------------------------
    # Salvar / Preview
    # ------------------------------------------------------------------

    def _on_save(self, _btn):
        if not self._current_theme or not self._theme_dir:
            return
        self.capture_z_order()
        path = os.path.join(self._theme_dir, "theme.yaml")
        try:
            self._current_theme.save(path)
            # Regenerate the home-screen preview from the current canvas so it
            # reflects the edited theme (background + all widgets in z-order).
            if self._canvas:
                try:
                    self._canvas.render_to_png(
                        os.path.join(self._theme_dir, "background.png"))
                except Exception as ex:
                    log.warning("Falha ao gerar preview: %s", ex)
            log.info("Tema salvo em %s", path)
            self._toast("Tema salvo com sucesso.")
        except Exception as e:
            self._show_error(f"Falha ao salvar:\n{e}")

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _show_error(self, msg: str):
        dialog = Adw.AlertDialog()
        dialog.set_heading("Erro")
        dialog.set_body(msg)
        dialog.add_response("ok", "OK")
        parent = self._editor_window or self._main_window
        if parent:
            dialog.present(parent)

    def _toast(self, msg: str):
        # Adw.Toast só funciona dentro de Adw.ToastOverlay — usar por enquanto log
        log.info("Toast: %s", msg)
