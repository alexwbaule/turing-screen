"""
System tray via D-Bus StatusNotifierItem (org.kde.StatusNotifierItem).
Funciona em KDE Plasma, XFCE, GNOME (com extensão AppIndicator).
Não usa GTK3 — sem conflito com GTK4.
"""
import logging
import os
from gi.repository import Gio, GLib

log = logging.getLogger(__name__)

_SNI_XML = """
<node>
  <interface name="org.kde.StatusNotifierItem">
    <property name="Category"      type="s"          access="read"/>
    <property name="Id"            type="s"          access="read"/>
    <property name="Title"         type="s"          access="read"/>
    <property name="Status"        type="s"          access="read"/>
    <property name="IconName"      type="s"          access="read"/>
    <property name="IconThemePath" type="s"          access="read"/>
    <property name="Menu"          type="o"          access="read"/>
    <property name="ToolTip"       type="(sa(iiay)ss)" access="read"/>
    <signal name="NewIcon"/>
    <signal name="NewStatus"><arg type="s"/></signal>
    <method name="Activate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="SecondaryActivate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="ContextMenu">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="Scroll">
      <arg name="delta"       type="i" direction="in"/>
      <arg name="orientation" type="s" direction="in"/>
    </method>
  </interface>
</node>
"""

_MENU_XML = """
<node>
  <interface name="com.canonical.dbusmenu">
    <property name="Version"        type="u"   access="read"/>
    <property name="TextDirection"  type="s"   access="read"/>
    <property name="Status"         type="s"   access="read"/>
    <property name="IconThemePath"  type="as"  access="read"/>
    <signal name="ItemsPropertiesUpdated">
      <arg type="a(ia{sv})" name="updatedProps"/>
      <arg type="a(ias)"    name="removedProps"/>
    </signal>
    <signal name="LayoutUpdated">
      <arg type="u" name="revision"/>
      <arg type="i" name="parent"/>
    </signal>
    <signal name="ItemActivationRequested">
      <arg type="i" name="id"/>
      <arg type="u" name="timestamp"/>
    </signal>
    <method name="GetLayout">
      <arg name="parentId"     type="i"          direction="in"/>
      <arg name="recursionDepth" type="i"         direction="in"/>
      <arg name="propertyNames" type="as"         direction="in"/>
      <arg name="revision"     type="u"          direction="out"/>
      <arg name="layout"       type="(ia{sv}av)" direction="out"/>
    </method>
    <method name="GetGroupProperties">
      <arg name="ids"           type="ai" direction="in"/>
      <arg name="propertyNames" type="as" direction="in"/>
      <arg name="properties"    type="a(ia{sv})" direction="out"/>
    </method>
    <method name="GetProperty">
      <arg name="id"       type="i"  direction="in"/>
      <arg name="name"     type="s"  direction="in"/>
      <arg name="value"    type="v"  direction="out"/>
    </method>
    <method name="Event">
      <arg name="id"        type="i" direction="in"/>
      <arg name="eventId"   type="s" direction="in"/>
      <arg name="data"      type="v" direction="in"/>
      <arg name="timestamp" type="u" direction="in"/>
    </method>
    <method name="EventGroup">
      <arg name="events"  type="a(isvu)" direction="in"/>
      <arg name="idErrors" type="ai"    direction="out"/>
    </method>
    <method name="AboutToShow">
      <arg name="id"         type="i" direction="in"/>
      <arg name="needUpdate" type="b" direction="out"/>
    </method>
    <method name="AboutToShowGroup">
      <arg name="ids"          type="ai" direction="in"/>
      <arg name="updatesNeeded" type="ai" direction="out"/>
      <arg name="idErrors"      type="ai" direction="out"/>
    </method>
  </interface>
</node>
"""


class TrayIcon:
    """
    Registra um StatusNotifierItem no D-Bus da sessão.

    Uso:
        tray = TrayIcon(
            icon_name="application-x-executable",
            title="Turing Screen",
            on_activate=lambda: show_window(),
            on_quit=lambda: app.quit(),
        )
        tray.start()   # non-blocking
        tray.stop()    # limpa a registro D-Bus
    """

    _WATCHER_BUS = "org.kde.StatusNotifierWatcher"
    _WATCHER_OBJ = "/StatusNotifierWatcher"
    _SNI_BUS_PREFIX = "org.kde.StatusNotifierItem"
    _SNI_OBJ = "/StatusNotifierItem"
    _MENU_OBJ = "/MenuBar"

    def __init__(self, icon_name: str, title: str,
                 on_activate=None, on_quit=None, on_quit_off=None,
                 on_context_menu=None):
        self._icon_name = icon_name
        self._title = title
        self._on_activate = on_activate
        self._on_quit = on_quit
        self._on_quit_off = on_quit_off
        self._on_context_menu = on_context_menu
        self._conn: Gio.DBusConnection | None = None
        self._sni_reg_id = 0
        self._menu_reg_id = 0
        self._bus_name_id = 0
        self._icon_theme_path = self._find_icon_theme_path()

    @staticmethod
    def _find_icon_theme_path() -> str:
        """Retorna o diretório de ícones do projeto se existir."""
        base = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
        p = os.path.join(base, "res", "icons")
        return p if os.path.isdir(p) else ""

    # ------------------------------------------------------------------
    # Inicialização
    # ------------------------------------------------------------------

    def start(self):
        """Registra o SNI no D-Bus (chamado na main thread GTK)."""
        try:
            self._conn = Gio.bus_get_sync(Gio.BusType.SESSION, None)
            sni_node = Gio.DBusNodeInfo.new_for_xml(_SNI_XML)
            menu_node = Gio.DBusNodeInfo.new_for_xml(_MENU_XML)

            # Registra o objeto SNI
            self._sni_reg_id = self._conn.register_object(
                self._SNI_OBJ,
                sni_node.interfaces[0],
                self._sni_method_call,
                self._sni_get_property,
                None,
            )
            # Registra o objeto de menu
            self._menu_reg_id = self._conn.register_object(
                self._MENU_OBJ,
                menu_node.interfaces[0],
                self._menu_method_call,
                self._menu_get_property,
                None,
            )

            # Solicita um nome de bus único
            self._bus_name_id = Gio.bus_own_name_on_connection(
                self._conn,
                f"{self._SNI_BUS_PREFIX}-{os.getpid()}",
                Gio.BusNameOwnerFlags.NONE,
                self._on_name_acquired,
                self._on_name_lost,
            )
        except Exception as e:
            log.warning("Tray: falha ao registrar no D-Bus: %s", e)

    def stop(self):
        if self._bus_name_id:
            Gio.bus_unown_name(self._bus_name_id)
        if self._conn and self._sni_reg_id:
            self._conn.unregister_object(self._sni_reg_id)
        if self._conn and self._menu_reg_id:
            self._conn.unregister_object(self._menu_reg_id)

    def _on_name_acquired(self, conn, name):
        """Nome D-Bus adquirido → registra no StatusNotifierWatcher."""
        try:
            conn.call(
                self._WATCHER_BUS,
                self._WATCHER_OBJ,
                self._WATCHER_BUS,
                "RegisterStatusNotifierItem",
                GLib.Variant("(s)", (name,)),
                None,
                Gio.DBusCallFlags.NONE,
                -1,
                None,
                None,
                None,
            )
            log.info("Tray: registrado no StatusNotifierWatcher como '%s'", name)
        except Exception as e:
            log.debug("Tray: RegisterStatusNotifierItem falhou: %s", e)

    def _on_name_lost(self, conn, name):
        log.debug("Tray: nome D-Bus perdido: %s", name)

    # ------------------------------------------------------------------
    # SNI interface handlers
    # ------------------------------------------------------------------

    def _sni_get_property(self, conn, sender, obj_path, iface, prop_name):
        if prop_name == "Category":
            return GLib.Variant("s", "ApplicationStatus")
        if prop_name == "Id":
            return GLib.Variant("s", "turing-screen")
        if prop_name == "Title":
            return GLib.Variant("s", self._title)
        if prop_name == "Status":
            return GLib.Variant("s", "Active")
        if prop_name == "IconName":
            return GLib.Variant("s", self._icon_name)
        if prop_name == "IconThemePath":
            return GLib.Variant("s", self._icon_theme_path)
        if prop_name == "Menu":
            return GLib.Variant("o", self._MENU_OBJ)
        if prop_name == "ToolTip":
            return GLib.Variant("(sa(iiay)ss)", (self._icon_name, [], self._title, ""))
        return None

    def _sni_method_call(self, conn, sender, obj_path, iface, method, params, invoc):
        log.debug("Tray SNI method=%s", method)
        if method in ("Activate", "SecondaryActivate"):
            if self._on_activate:
                GLib.idle_add(self._on_activate)
        elif method == "ContextMenu":
            x, y = params[0], params[1]
            log.info("Tray: ContextMenu(%d, %d)", x, y)
            if self._on_context_menu:
                GLib.idle_add(self._on_context_menu, x, y)
        invoc.return_value(None)

    # ------------------------------------------------------------------
    # dbusmenu handlers (menu contextual)
    # ------------------------------------------------------------------

    # Itens do menu: (id, label, acção)  — id=0 é reservado para o root, nunca usar aqui
    _MENU_ITEMS = [
        (1,  "Mostrar / Ocultar",    "activate"),
        (10, "",                     "separator"),
        (2,  "Finalizar",            "quit"),
        (3,  "Finalizar e Desligar", "quit_off"),
    ]

    def _menu_get_property(self, conn, sender, obj_path, iface, prop_name):
        if prop_name == "Version":
            return GLib.Variant("u", 3)
        if prop_name == "TextDirection":
            return GLib.Variant("s", "ltr")
        if prop_name == "Status":
            return GLib.Variant("s", "normal")
        if prop_name == "IconThemePath":
            return GLib.Variant("as", [])
        return None

    def _menu_method_call(self, conn, sender, obj_path, iface, method, params, invoc):
        log.debug("Tray dbusmenu method=%s", method)
        if method == "GetLayout":
            revision, layout = self._build_menu_layout()
            invoc.return_value(GLib.Variant("(u(ia{sv}av))", (revision, layout)))
        elif method == "GetGroupProperties":
            invoc.return_value(GLib.Variant("(a(ia{sv}))", ([],)))
        elif method == "GetProperty":
            invoc.return_value(GLib.Variant("(v)", (GLib.Variant("s", ""),)))
        elif method == "Event":
            item_id = params[0]
            event_id = params[1]
            log.debug("Tray dbusmenu Event id=%d event=%s", item_id, event_id)
            if event_id in ("clicked", "activated"):
                self._handle_menu_click(item_id)
            invoc.return_value(GLib.Variant("()", ()))
        elif method == "EventGroup":
            for ev in params[0]:
                if ev[1] == "clicked":
                    self._handle_menu_click(ev[0])
            invoc.return_value(GLib.Variant("(ai)", ([],)))
        elif method == "AboutToShow":
            invoc.return_value(GLib.Variant("(b)", (False,)))
        elif method == "AboutToShowGroup":
            invoc.return_value(GLib.Variant("(aiai)", ([], [])))
        else:
            invoc.return_value(GLib.Variant("()", ()))

    def _handle_menu_click(self, item_id: int):
        if item_id == 1 and self._on_activate:
            GLib.idle_add(self._on_activate)
        elif item_id == 2 and self._on_quit:
            GLib.idle_add(self._on_quit)
        elif item_id == 3 and self._on_quit_off:
            GLib.idle_add(self._on_quit_off)

    def _build_menu_layout(self) -> tuple:
        """Constrói layout dbusmenu.

        Retorna (revision, layout_tuple) onde layout_tuple é uma tupla Python
        compatível com o formato `(ia{sv}av)` esperado pelo GetLayout.

        Cada child em `av` é um GLib.Variant("(ia{sv}av)") — o PyGObject
        auto-embala como `v` ao serializar o array `av`.
        Props das raiz e dos items são dicts Python simples (não GLib.Variant
        pré-construídos) para o PyGObject inferir os tipos corretamente.
        """
        children = []
        for item_id, label, action in self._MENU_ITEMS:
            if action == "separator":
                props = {"type": GLib.Variant("s", "separator")}
            else:
                props = {
                    "label":   GLib.Variant("s", label),
                    "enabled": GLib.Variant("b", True),
                    "visible": GLib.Variant("b", True),
                }
            children.append(GLib.Variant("(ia{sv}av)", (item_id, props, [])))
        # Root usa dict vazio — não GLib.Variant pré-construído
        return (1, (0, {}, children))
