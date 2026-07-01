"""
Properties panel — shows editable fields for the selected canvas element.
Mirrors properties.go from the Fyne version with full field parity.
"""
import dataclasses as _dc
import os

from gi.repository import Gtk, Gdk, Pango, Gio
from i18n import _

# Suffix tokens that identify the representation type in a yaml_path
# (same set as in canvas.py — kept here to avoid a circular import).
_KNOWN_SUFFIXES = frozenset({
    "TEXT", "PERCENT_TEXT", "GRAPH", "RADIAL", "CHART", "GAUGE", "STATUS_BAR",
})

# Maps each type-kind string to the attribute name it occupies on a Measurement.
_SLOT_FOR_KIND = {
    "text":         "Text",
    "percent_text": "PercentText",
    "graph":        "Graph",
    "radial":       "Radial",
    "chart":        "Chart",
    "gauge":        "Gauge",
    "status_bar":   "StatusBar",
}
# Reverse map: slot attr name → kind string (for determining current_type).
_SLOT_TO_KIND = {v: k for k, v in _SLOT_FOR_KIND.items()}

# Allowed kind sets matching what the daemon compositor actually renders.
_FLOAT_KINDS = frozenset({"text", "graph", "radial", "chart", "gauge", "status_bar"})


def _kinds_allowed_for_path(yaml_path: str):
    """Return frozenset of kind strings the daemon actually renders for this path.

    Returns None when all types are valid (Sensor with collectMeasurementItems).
    Only restricts sensors where the compositor uses collectMeasurementFloatItems
    (custom text formatter — PercentText doesn't apply there)."""
    p = yaml_path
    # Network and Frequency use collectMeasurementFloatItems (custom text formatter).
    # PercentText is excluded — "percent of bytes/sec" has no meaning.
    if (p.startswith("STATS.NET.ETH.") or p.startswith("STATS.NET.WLO.")
            or p.startswith("STATS.CPU.FREQUENCY.") or p.startswith("STATS.GPU.FREQUENCY.")):
        return _FLOAT_KINDS
    return None


# ── Color helpers ──────────────────────────────────────────────────────────────

def _str_to_rgba(s: str, default: str = "0,0,0") -> Gdk.RGBA:
    """Parse 'R,G,B' / '#RRGGBB' → Gdk.RGBA."""
    s = (s or "").strip()
    if not s:
        s = default
    rgba = Gdk.RGBA()
    if "," in s:
        try:
            parts = [int(x.strip()) for x in s.split(",")]
            rgba.red   = parts[0] / 255.0
            rgba.green = parts[1] / 255.0
            rgba.blue  = parts[2] / 255.0
            rgba.alpha = parts[3] / 255.0 if len(parts) == 4 else 1.0
            return rgba
        except (ValueError, IndexError):
            pass
    if s.startswith("#"):
        rgba.parse(s)
        return rgba
    rgba.red = rgba.green = rgba.blue = 0.0
    rgba.alpha = 1.0
    return rgba


def _rgba_to_str(rgba: Gdk.RGBA) -> str:
    return f"{round(rgba.red*255)},{round(rgba.green*255)},{round(rgba.blue*255)}"


# ──────────────────────────────────────────────────────────────────────────────

class PropertiesPanel(Gtk.Box):

    def __init__(self):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        self.set_size_request(280, -1)
        self._current = None
        self._updating = False
        self._on_delete = None
        self._on_type_change = None
        self._theme_dir = ""
        self._build_ui()

    def set_on_delete(self, cb):
        self._on_delete = cb

    def set_theme_dir(self, theme_dir: str):
        """Theme directory (set on load) — used to resolve font file paths."""
        self._theme_dir = theme_dir or ""

    def set_on_type_change(self, cb):
        self._on_type_change = cb

    # ── UI skeleton ────────────────────────────────────────────────────────────

    def _build_ui(self):
        self._header = Gtk.Label(label=_("Properties"))
        self._header.add_css_class("heading")
        self._header.set_halign(Gtk.Align.START)
        self._header.set_margin_top(8)
        self._header.set_margin_start(8)
        self._header.set_margin_bottom(4)
        self.append(self._header)
        self.append(Gtk.Separator())

        scroll = Gtk.ScrolledWindow()
        scroll.set_vexpand(True)
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
        # Non-overlay scrollbar: it reserves its own column instead of floating
        # over the content, so it no longer covers each SpinButton's +/- steppers.
        scroll.set_overlay_scrolling(False)

        self._content = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
        self._content.set_margin_top(8)
        self._content.set_margin_start(8)
        self._content.set_margin_end(8)
        self._content.set_margin_bottom(8)
        scroll.set_child(self._content)
        self.append(scroll)

        ph = Gtk.Label(label=_("Select an element on the canvas"))
        ph.set_sensitive(False)
        ph.set_halign(Gtk.Align.CENTER)
        ph.set_valign(Gtk.Align.CENTER)
        ph.set_vexpand(True)
        self._content.append(ph)
        self._placeholder = ph

    # ── Public API ─────────────────────────────────────────────────────────────

    def update(self, element):
        self._current = element
        self._clear_content()

        if element is None:
            self._header.set_label(_("Properties"))
            self._content.append(self._placeholder)
            return

        from ui.draggable import (DraggableText, DraggableGraph, DraggableRadial,
                                   DraggableChart, DraggableGauge, DraggableStatusBar,
                                   DraggableImage)
        self._header.set_label(f"Properties — {element.yaml_path.split('.')[-1]}")

        # Common header: path info + delete
        self._build_common_header(element)

        if isinstance(element, DraggableText):
            self._build_text_props(element)
        elif isinstance(element, DraggableGraph):
            self._build_graph_props(element)
        elif isinstance(element, DraggableRadial):
            self._build_radial_props(element)
        elif isinstance(element, DraggableChart):
            self._build_chart_props(element)
        elif isinstance(element, DraggableGauge):
            self._build_gauge_props(element)
        elif isinstance(element, DraggableStatusBar):
            self._build_statusbar_props(element)
        elif isinstance(element, DraggableImage):
            self._build_image_props(element)

    def update_position(self, element):
        if element is not self._current:
            return
        x, y = element.get_model_position()
        if hasattr(self, "_x_spin"):
            self._updating = True
            self._x_spin.set_value(x)
            self._y_spin.set_value(y)
            self._updating = False

    def update_bg_image(self, key: str, img, refresh_cb):
        """Show editable properties for a background image (StaticImage)."""
        self._current = None
        self._clear_content()
        self._header.set_label(f"Background — {key}")

        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=4)
        box.set_margin_bottom(6)
        path_lbl = Gtk.Label(label=f"{key}  ({os.path.basename(img.PATH or '')})")
        path_lbl.set_halign(Gtk.Align.START)
        path_lbl.set_ellipsize(Pango.EllipsizeMode.START)
        path_lbl.add_css_class("dim-label")
        path_lbl.set_selectable(True)
        box.append(path_lbl)
        self._content.append(box)

        exp, inner = self._expander(_("Position & Size"))
        self._x_spin = self._spin(img.X or 0, -9999, 9999,
                                  lambda s: [setattr(img, "X", int(s.get_value())), refresh_cb()])
        self._y_spin = self._spin(img.Y or 0, -9999, 9999,
                                  lambda s: [setattr(img, "Y", int(s.get_value())), refresh_cb()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._append_row(inner, _("Width"),
                         self._spin(img.WIDTH or 1, 1, 9999,
                                    lambda s: [setattr(img, "WIDTH", int(s.get_value())), refresh_cb()]))
        self._append_row(inner, _("Height"),
                         self._spin(img.HEIGHT or 1, 1, 9999,
                                    lambda s: [setattr(img, "HEIGHT", int(s.get_value())), refresh_cb()]))
        self._content.append(exp)

    # ── Widget factories ───────────────────────────────────────────────────────

    def _clear_content(self):
        child = self._content.get_first_child()
        while child:
            nxt = child.get_next_sibling()
            self._content.remove(child)
            child = nxt

    def _row(self, label: str, widget: Gtk.Widget, expand: bool = True) -> Gtk.Box:
        row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        lbl = Gtk.Label(label=label)
        lbl.set_width_chars(13)
        lbl.set_halign(Gtk.Align.END)
        lbl.add_css_class("dim-label")
        row.append(lbl)
        if expand:
            widget.set_hexpand(True)
        row.append(widget)
        return row

    def _spin(self, value, lo, hi, changed_cb, step=1.0) -> Gtk.SpinButton:
        adj = Gtk.Adjustment(value=value, lower=lo, upper=hi, step_increment=step)
        spin = Gtk.SpinButton(adjustment=adj)
        spin.set_numeric(True)
        spin.connect("value-changed", changed_cb)
        return spin

    def _entry(self, text: str, changed_cb) -> Gtk.Entry:
        e = Gtk.Entry()
        e.set_text(text or "")
        e.connect("changed", changed_cb)
        return e

    def _font_row(self, parent: Gtk.Box, data, refresh_cb):
        """Font-file row: editable path + a browse button that opens a file
        picker (.otf/.ttf). The theme stores a path relative to the theme dir
        (e.g. 'turtheme/Azonix.otf'), so the picked file is relativised."""
        row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        lbl = Gtk.Label(label=_("Font file"))
        lbl.set_width_chars(13)
        lbl.set_halign(Gtk.Align.END)
        lbl.add_css_class("dim-label")
        entry = Gtk.Entry()
        entry.set_text(data.FONT or "")
        entry.set_hexpand(True)
        entry.connect("changed", lambda e: [setattr(data, "FONT", e.get_text()), refresh_cb()])
        browse = Gtk.Button()
        browse.set_icon_name("document-open-symbolic")
        browse.set_tooltip_text(_("Browse font…"))
        browse.connect("clicked", lambda b: self._pick_font(entry, data, refresh_cb))
        row.append(lbl)
        row.append(entry)
        row.append(browse)
        parent.append(row)

    def _fonts_dir(self) -> str:
        """Font files live at <repo>/res/fonts (the daemon loads them as
        FONTPATH + FONT). Derive it from this module's location so it's right
        regardless of cwd or theme_dir."""
        root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
        return os.path.join(root, "res", "fonts")

    def _pick_font(self, entry: Gtk.Entry, data, refresh_cb):
        dialog = Gtk.FileDialog()
        dialog.set_title(_("Select font"))
        filt = Gtk.FileFilter()
        filt.set_name(_("Font files (*.otf, *.ttf)"))
        for pat in ("*.otf", "*.ttf", "*.ttc"):
            filt.add_pattern(pat)
        filters = Gio.ListStore.new(Gtk.FileFilter)
        filters.append(filt)
        dialog.set_filters(filters)
        # Open at res/fonts (FONT is stored relative to it).
        fonts_dir = self._fonts_dir()
        if os.path.isdir(fonts_dir):
            try:
                dialog.set_initial_folder(Gio.File.new_for_path(fonts_dir))
            except Exception:
                pass

        def _done(d, res):
            try:
                gfile = d.open_finish(res)
            except Exception:
                return  # cancelled
            p = gfile.get_path()
            # FONT is relative to res/fonts (e.g. 'turtheme/Azonix.otf').
            try:
                p = os.path.relpath(p, fonts_dir)
            except ValueError:
                pass
            # set_text fires "changed" → updates data.FONT + refresh.
            entry.set_text(p)

        dialog.open(entry.get_root(), None, _done)

    def _color_btn(self, color_str, on_color, default: str = "0,0,0") -> Gtk.ColorButton:
        btn = Gtk.ColorButton()
        btn.set_use_alpha(False)
        btn.set_rgba(_str_to_rgba(color_str or "", default))
        btn.set_hexpand(True)
        btn.connect("color-set", lambda b: on_color(_rgba_to_str(b.get_rgba())))
        return btn

    def _optional_color_row(self, parent: Gtk.Box, label: str, data, attr: str,
                             default: str = "0,0,0", refresh_cb=None):
        """Row with [☐ ColorButton].  Unchecked → attr = '' (ignored by daemon)."""
        current = (getattr(data, attr, "") or "").strip()
        enabled = bool(current)

        btn = Gtk.ColorButton()
        btn.set_use_alpha(False)
        btn.set_rgba(_str_to_rgba(current if enabled else default, default))
        btn.set_hexpand(True)
        btn.set_sensitive(enabled)

        chk = Gtk.CheckButton()
        chk.set_active(enabled)
        chk.set_tooltip_text(_("Enable this color (uncheck → ignored)"))

        def on_color(_b):
            if chk.get_active():
                setattr(data, attr, _rgba_to_str(btn.get_rgba()))
                if refresh_cb:
                    refresh_cb()

        def on_toggle(_b):
            active = _b.get_active()
            btn.set_sensitive(active)
            setattr(data, attr, _rgba_to_str(btn.get_rgba()) if active else "")
            if refresh_cb:
                refresh_cb()

        btn.connect("color-set", on_color)
        chk.connect("toggled", on_toggle)

        inner = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=4)
        inner.append(chk)
        inner.append(btn)
        parent.append(self._row(label, inner))

    def _toggle(self, active: bool, label: str, changed_cb) -> Gtk.CheckButton:
        cb = Gtk.CheckButton(label=label)
        cb.set_active(bool(active))
        cb.connect("toggled", lambda b: changed_cb(b.get_active()))
        return cb

    def _dropdown(self, options: list[str], current: str, changed_cb) -> Gtk.DropDown:
        current = (current or options[0]).upper()
        dd = Gtk.DropDown.new_from_strings(options)
        try:
            dd.set_selected(options.index(current))
        except ValueError:
            dd.set_selected(0)
        dd.connect("notify::selected", lambda d, _: changed_cb(options[d.get_selected()])
                   if not self._updating else None)
        return dd

    def _expander(self, title: str, expanded: bool = True) -> tuple[Gtk.Expander, Gtk.Box]:
        """Return (expander, inner_box). Append rows to inner_box."""
        exp = Gtk.Expander(label=title)
        exp.set_expanded(expanded)
        exp.set_margin_top(4)
        inner = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
        inner.set_margin_top(4)
        inner.set_margin_start(8)
        inner.set_margin_bottom(4)
        exp.set_child(inner)
        return exp, inner

    def _section_label(self, parent: Gtk.Box, title: str):
        lbl = Gtk.Label(label=title)
        lbl.add_css_class("heading")
        lbl.set_halign(Gtk.Align.START)
        lbl.set_margin_top(6)
        parent.append(lbl)
        parent.append(Gtk.Separator())

    def _append_row(self, parent: Gtk.Box, label: str, widget: Gtk.Widget, expand=True):
        parent.append(self._row(label, widget, expand))

    # ── Common header ──────────────────────────────────────────────────────────

    def _build_common_header(self, element):
        from ui.draggable import (DraggableText, DraggableGraph, DraggableRadial,
                                   DraggableChart, DraggableGauge, DraggableStatusBar,
                                   DraggableImage)

        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=4)
        box.set_margin_bottom(6)

        # Full YAML path label
        path_lbl = Gtk.Label(label=element.yaml_path)
        path_lbl.set_halign(Gtk.Align.START)
        path_lbl.set_ellipsize(Pango.EllipsizeMode.START)
        path_lbl.add_css_class("dim-label")
        path_lbl.set_selectable(True)
        box.append(path_lbl)

        # Type switcher dropdown (not for images — they aren't type-swappable).
        #
        # Shown only when the element has a known parent measurement and its
        # yaml_path ends with a recognised type suffix (TEXT, GRAPH, …).
        # Elements in text-only terminal slots (USED, FREE, CONDITION, etc.)
        # or without a parent measurement (Load, Volume, Weather condition)
        # hide the dropdown because _replace_suffix would generate wrong paths.
        #
        # The option list is filtered to slots that actually exist on the
        # Measurement dataclass and are allowed for this sensor path.
        if not isinstance(element, DraggableImage):
            _type_map = [
                (DraggableText,      "text"),
                (DraggableGraph,     "graph"),
                (DraggableRadial,    "radial"),
                (DraggableChart,     "chart"),
                (DraggableGauge,     "gauge"),
                (DraggableStatusBar, "status_bar"),
            ]

            # Determine current type from _slot first (most reliable), then
            # fall back to yaml_path suffix / Python class.
            slot = getattr(element, "_slot", None)
            if slot and slot in _SLOT_TO_KIND:
                current_type = _SLOT_TO_KIND[slot]
            elif isinstance(element, DraggableText) and element.yaml_path.endswith(".PERCENT_TEXT"):
                current_type = "percent_text"
            else:
                current_type = next((t for cls, t in _type_map if isinstance(element, cls)), "text")

            _all_opts = [
                ("Text",       "text"),
                ("% Text",     "percent_text"),
                ("Graph",      "graph"),
                ("Radial",     "radial"),
                ("Chart",      "chart"),
                ("Gauge",      "gauge"),
                ("Status Bar", "status_bar"),
            ]

            # Decide whether to show the dropdown and which options to include.
            meas = getattr(element, "_measurement", None)
            path_terminal = element.yaml_path.rsplit(".", 1)[-1] if "." in element.yaml_path else ""

            show_type_dd = (
                meas is not None
                and path_terminal in _KNOWN_SUFFIXES
            )

            if show_type_dd:
                if _dc.is_dataclass(meas):
                    meas_fields = {f.name for f in _dc.fields(meas)}
                    opts = [(lbl, k) for lbl, k in _all_opts
                            if _SLOT_FOR_KIND.get(k) in meas_fields]
                    # Always keep the current type in the list even if the slot
                    # is not in _SLOT_FOR_KIND (safety fallback).
                    if not opts or not any(k == current_type for _, k in opts):
                        opts = _all_opts
                else:
                    opts = _all_opts

                # Further restrict to what the daemon compositor actually renders
                # for this sensor path (e.g. Date/Net → Text only; Frequency →
                # no Gauge/StatusBar/PercentText; Memory → no Gauge/Chart/Text).
                allowed_kinds = _kinds_allowed_for_path(element.yaml_path)
                if allowed_kinds is not None:
                    filtered = [(lbl, k) for lbl, k in opts if k in allowed_kinds]
                    if filtered:
                        opts = filtered

                labels = [lbl for lbl, _ in opts]
                kinds  = [k   for _, k  in opts]

                dd = Gtk.DropDown.new_from_strings(labels)
                dd.set_hexpand(True)
                self._updating = True
                try:
                    dd.set_selected(kinds.index(current_type))
                except ValueError:
                    dd.set_selected(0)
                self._updating = False

                def _on_type_selected(d, _, _kinds=kinds, _cur=current_type):
                    if self._updating:
                        return
                    new_type = _kinds[d.get_selected()]
                    if new_type != _cur and self._on_type_change:
                        self._on_type_change(element, new_type)

                dd.connect("notify::selected", _on_type_selected)

                type_row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
                lbl = Gtk.Label(label=_("Type"))
                lbl.set_width_chars(13)
                lbl.set_halign(Gtk.Align.END)
                lbl.add_css_class("dim-label")
                type_row.append(lbl)
                type_row.append(dd)
                box.append(type_row)

        # Delete button
        del_btn = Gtk.Button(label=_("Delete element"))
        del_btn.add_css_class("destructive-action")
        del_btn.connect("clicked", lambda _: self._on_delete and self._on_delete(element))
        box.append(del_btn)

        self._content.append(box)
        self._content.append(Gtk.Separator())

    # ── Image (non-BACKGROUND static image) ────────────────────────────────────

    def _build_image_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def resize():
            elem.widget.set_size_request(data.WIDTH or 100, data.HEIGHT or 100)
            refresh()

        exp, inner = self._expander(_("Position & Size"))
        self._x_spin = self._spin(data.X, -9999, 9999, lambda s: [setattr(data, "X", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._y_spin = self._spin(data.Y, -9999, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._append_row(inner, _("Width"),  self._spin(data.WIDTH or 100,  1, 9999, lambda s: [setattr(data, "WIDTH", int(s.get_value())), resize()]))
        self._append_row(inner, _("Height"), self._spin(data.HEIGHT or 100, 1, 9999, lambda s: [setattr(data, "HEIGHT", int(s.get_value())), resize()]))
        self._content.append(exp)

    # ── Text ───────────────────────────────────────────────────────────────────

    def _build_text_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        # Position
        exp, inner = self._expander(_("Position"))
        # data.X is the ALIGN anchor (left/center/right edge), not the widget
        # top-left — so we must NOT set_position(data.X, data.Y) here (that would
        # place the left edge at the anchor and shift the text on every release).
        # refresh() → invalidate() re-anchors via _anchor_to_left, and is guarded
        # by _updating so update_position's programmatic set_value won't move it.
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), refresh()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), refresh()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._content.append(exp)

        # Content
        exp, inner = self._expander(_("Content"))
        self._append_row(inner, _("Text"), self._entry(data.TEXT or data.PLACEHOLDER, lambda e: [setattr(data, "TEXT", e.get_text()), refresh()]))
        self._append_row(inner, _("Format"), self._entry(data.FORMAT or "", lambda e: [setattr(data, "FORMAT", e.get_text()), refresh()]))
        inner.append(self._toggle(bool(data.SHOW_UNIT), _("Show unit"), lambda v: [setattr(data, "SHOW_UNIT", v), refresh()]))
        self._content.append(exp)

        # Font
        exp, inner = self._expander(_("Font"))
        self._font_row(inner, data, refresh)
        self._append_row(inner, _("Font color"), self._color_btn(data.FONT_COLOR, lambda c: [setattr(data, "FONT_COLOR", c), refresh()], "255,255,255"), expand=False)
        self._optional_color_row(inner, "Background", data, "BACKGROUND_COLOR", "0,0,0", refresh)
        self._content.append(exp)

        # Layout
        exp, inner = self._expander(_("Layout"))
        self._append_row(inner, _("Align"), self._dropdown(["LEFT", "CENTER", "RIGHT"], data.ALIGN or "LEFT", lambda v: [setattr(data, "ALIGN", v), refresh()]))
        self._content.append(exp)

    # ── Graph ──────────────────────────────────────────────────────────────────

    def _build_graph_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def resize():
            elem.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 80)
            refresh()

        # Position & Size
        exp, inner = self._expander(_("Position & Size"))
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._graph_w_spin = self._spin(data.WIDTH or 200,  10, 9999, lambda s: [setattr(data, "WIDTH", int(s.get_value())), resize()])
        self._graph_h_spin = self._spin(data.HEIGHT or 80, 10, 9999, lambda s: [setattr(data, "HEIGHT", int(s.get_value())), resize()])
        self._append_row(inner, _("Width"),  self._graph_w_spin)
        self._append_row(inner, _("Height"), self._graph_h_spin)

        def _on_direction(v):
            horiz = lambda d: (d or "left").lower() in ("left", "right")
            old = data.DIRECTION
            data.DIRECTION = v
            # Flip W/H when crossing horizontal <-> vertical so a wide-short
            # bar becomes a narrow-tall one (and vice-versa).
            if horiz(old) != horiz(v):
                data.WIDTH, data.HEIGHT = data.HEIGHT, data.WIDTH
                self._updating = True
                self._graph_w_spin.set_value(data.WIDTH or 200)
                self._graph_h_spin.set_value(data.HEIGHT or 80)
                self._updating = False
                elem.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 80)
            refresh()

        self._append_row(inner, _("Direction"), self._dropdown(["LEFT", "RIGHT", "UP", "DOWN"], data.DIRECTION or "LEFT", _on_direction))
        self._content.append(exp)

        # Appearance
        exp, inner = self._expander(_("Appearance"))
        self._append_row(inner, _("Bar color"), self._color_btn(data.BAR_COLOR, lambda c: [setattr(data, "BAR_COLOR", c), refresh()], "0,255,0"), expand=False)
        self._optional_color_row(inner, "Background", data, "BACKGROUND_COLOR", "26,26,26", refresh)
        self._optional_color_row(inner, _("Gradient"),   data, "GRADIENT_COLOR",   "0,0,0",   refresh)
        inner.append(self._toggle(bool(data.BAR_OUTLINE),  _("Bar outline"),  lambda v: [setattr(data, "BAR_OUTLINE", v), refresh()]))
        inner.append(self._toggle(bool(data.REVERT_VALUE), _("Revert value"), lambda v: [setattr(data, "REVERT_VALUE", v), refresh()]))
        self._content.append(exp)

        # Range
        exp, inner = self._expander(_("Range"))
        self._append_row(inner, _("Min"), self._spin(data.MIN_VALUE or 0,   -9999, 9999, lambda s: [setattr(data, "MIN_VALUE", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Max"), self._spin(data.MAX_VALUE or 100, -9999, 9999, lambda s: [setattr(data, "MAX_VALUE", int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Advanced
        exp, inner = self._expander(_("Advanced"), expanded=False)
        self._append_row(inner, _("Steps"),         self._spin(data.STEPS or 0,         0, 500, lambda s: [setattr(data, "STEPS",         int(s.get_value())), refresh()]))
        self._append_row(inner, _("Step gap"),      self._spin(data.STEP_GAP or 0,      0, 100, lambda s: [setattr(data, "STEP_GAP",      int(s.get_value())), refresh()]))
        self._append_row(inner, _("Block width"),   self._spin(data.BLOCK_WIDTH or 0,   0, 200, lambda s: [setattr(data, "BLOCK_WIDTH",   int(s.get_value())), refresh()]))
        self._append_row(inner, _("Corner radius"), self._spin(data.CORNER_RADIUS or 0, 0, 100, lambda s: [setattr(data, "CORNER_RADIUS", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Border width"),  self._spin(data.BORDER_WIDTH or 0,  0, 20,  lambda s: [setattr(data, "BORDER_WIDTH",  int(s.get_value())), refresh()]))
        self._content.append(exp)

    # ── Radial ─────────────────────────────────────────────────────────────────

    def _build_radial_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def _move():
            r = data.RADIUS or 80
            elem.set_position(data.X - r, data.Y - r)
            refresh()

        def _resize_radius(v):
            setattr(data, "RADIUS", v)
            elem.widget.set_size_request(v * 2, v * 2)
            _move()

        # Position
        exp, inner = self._expander(_("Position"))
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), _move()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), _move()])
        self._append_row(inner, _("X (center)"), self._x_spin)
        self._append_row(inner, _("Y (center)"), self._y_spin)
        self._content.append(exp)

        # Geometry
        exp, inner = self._expander(_("Geometry"))
        self._append_row(inner, _("Radius"), self._spin(data.RADIUS or 80,  10, 500, lambda s: _resize_radius(int(s.get_value()))))
        self._append_row(inner, _("Width"),  self._spin(data.WIDTH  or 10,   1, 200, lambda s: [setattr(data, "WIDTH", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Min"),    self._spin(data.MIN_VALUE or 0,   -9999, 9999, lambda s: [setattr(data, "MIN_VALUE", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Max"),    self._spin(data.MAX_VALUE or 100, -9999, 9999, lambda s: [setattr(data, "MAX_VALUE", int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Angles
        exp, inner = self._expander(_("Angles"))
        self._append_row(inner, _("Start °"),      self._spin(data.ANGLE_START or 0,   -360, 360, lambda s: [setattr(data, "ANGLE_START",  int(s.get_value())), refresh()]))
        self._append_row(inner, _("End °"),        self._spin(data.ANGLE_END   or 60,  -360, 360, lambda s: [setattr(data, "ANGLE_END",    int(s.get_value())), refresh()]))
        self._append_row(inner, _("Seg. count"),   self._spin(data.ANGLE_STEPS or 0,   0, 360, lambda s: [setattr(data, "ANGLE_STEPS",  int(s.get_value())), refresh()]))
        self._append_row(inner, _("Seg. sep °"),   self._spin(data.ANGLE_SEP   or 0,   0, 30,  lambda s: [setattr(data, "ANGLE_SEP",    int(s.get_value())), refresh()]))
        self._append_row(inner, _("Block angle °"),self._spin(data.BLOCK_ANGLE or 0,   0, 360, lambda s: [setattr(data, "BLOCK_ANGLE",  int(s.get_value())), refresh()]))
        inner.append(self._toggle(bool(data.CLOCKWISE),    _("Clockwise"),    lambda v: [setattr(data, "CLOCKWISE",    v), refresh()]))
        inner.append(self._toggle(bool(data.ROUND),        _("Round caps"),   lambda v: [setattr(data, "ROUND",        v), refresh()]))
        inner.append(self._toggle(bool(data.REVERT),       _("Revert colors"),lambda v: [setattr(data, "REVERT",       v), refresh()]))
        inner.append(self._toggle(bool(data.REVERT_VALUE), _("Revert value"), lambda v: [setattr(data, "REVERT_VALUE", v), refresh()]))
        self._content.append(exp)

        # Appearance
        exp, inner = self._expander(_("Appearance"))
        self._append_row(inner, _("Bar color"), self._color_btn(data.BAR_COLOR, lambda c: [setattr(data, "BAR_COLOR", c), refresh()], "0,255,0"), expand=False)
        self._optional_color_row(inner, "Background", data, "BACKGROUND_COLOR", "26,26,26", refresh)
        self._optional_color_row(inner, _("Gradient"),   data, "GRADIENT_COLOR",   "0,0,0",   refresh)
        self._content.append(exp)

        # Text inside ring
        exp, inner = self._expander(_("Center text"), expanded=False)
        inner.append(self._toggle(bool(data.SHOW_TEXT), _("Show text"),  lambda v: [setattr(data, "SHOW_TEXT", v), refresh()]))
        inner.append(self._toggle(bool(data.SHOW_UNIT), _("Show unit"),  lambda v: [setattr(data, "SHOW_UNIT", v), refresh()]))
        self._font_row(inner, data, refresh)
        self._append_row(inner, _("Font color"), self._color_btn(data.FONT_COLOR, lambda c: [setattr(data, "FONT_COLOR", c), refresh()], "255,255,255"), expand=False)
        self._content.append(exp)

    # ── Chart ──────────────────────────────────────────────────────────────────

    def _build_chart_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def resize():
            elem.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 100)
            refresh()

        # Position & Size
        exp, inner = self._expander(_("Position & Size"))
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._append_row(inner, _("Width"),  self._spin(data.WIDTH  or 200, 10, 9999, lambda s: [setattr(data, "WIDTH",  int(s.get_value())), resize()]))
        self._append_row(inner, _("Height"), self._spin(data.HEIGHT or 100, 10, 9999, lambda s: [setattr(data, "HEIGHT", int(s.get_value())), resize()]))
        self._content.append(exp)

        # Range
        exp, inner = self._expander(_("Range"))
        self._append_row(inner, _("Min"), self._spin(data.MIN_VALUE or 0,   -9999, 9999, lambda s: [setattr(data, "MIN_VALUE", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Max"), self._spin(data.MAX_VALUE or 100, -9999, 9999, lambda s: [setattr(data, "MAX_VALUE", int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Appearance
        exp, inner = self._expander(_("Appearance"))
        self._append_row(inner, _("Style"), self._dropdown(["BAR", "LINE"], data.STYLE or "BAR", lambda v: [setattr(data, "STYLE", v), refresh()]))
        self._append_row(inner, _("Col. width"), self._spin(data.COLUMN_WIDTH or 4,  1, 100, lambda s: [setattr(data, "COLUMN_WIDTH", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Col. gap"),   self._spin(data.COLUMN_GAP   or 1,  0, 50,  lambda s: [setattr(data, "COLUMN_GAP",   int(s.get_value())), refresh()]))
        self._append_row(inner, _("Fill color"), self._color_btn(data.FILL_COLOR, lambda c: [setattr(data, "FILL_COLOR", c), refresh()], "0,204,0"), expand=False)
        self._append_row(inner, _("Line color"), self._color_btn(data.LINE_COLOR, lambda c: [setattr(data, "LINE_COLOR", c), refresh()], "0,255,0"), expand=False)
        self._append_row(inner, _("Border width"), self._spin(data.BORDER_WIDTH or 0, 0, 20, lambda s: [setattr(data, "BORDER_WIDTH", int(s.get_value())), refresh()]))
        self._content.append(exp)

    # ── Gauge ──────────────────────────────────────────────────────────────────

    def _build_gauge_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def _move():
            r = data.RADIUS or 80
            elem.set_position(data.X - r, data.Y - r)
            refresh()

        def _resize_radius(v):
            setattr(data, "RADIUS", v)
            elem.widget.set_size_request(v * 2, v * 2)
            _move()

        # Position
        exp, inner = self._expander(_("Position"))
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), _move()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), _move()])
        self._append_row(inner, _("X (center)"), self._x_spin)
        self._append_row(inner, _("Y (center)"), self._y_spin)
        self._content.append(exp)

        # Geometry
        exp, inner = self._expander(_("Geometry"))
        self._append_row(inner, _("Radius"),       self._spin(data.RADIUS or 80,       10, 500,  lambda s: _resize_radius(int(s.get_value()))))
        self._append_row(inner, _("Needle width"), self._spin(data.NEEDLE_WIDTH or 4,  1,  20,   lambda s: [setattr(data, "NEEDLE_WIDTH", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Min"),          self._spin(data.MIN_VALUE or 0,   -9999, 9999, lambda s: [setattr(data, "MIN_VALUE", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Max"),          self._spin(data.MAX_VALUE or 100, -9999, 9999, lambda s: [setattr(data, "MAX_VALUE", int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Angles
        exp, inner = self._expander(_("Angles"))
        self._append_row(inner, _("Start °"), self._spin(data.ANGLE_START or -210, -360, 360, lambda s: [setattr(data, "ANGLE_START", int(s.get_value())), refresh()]))
        self._append_row(inner, _("End °"),   self._spin(data.ANGLE_END   or  30,  -360, 360, lambda s: [setattr(data, "ANGLE_END",   int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Appearance
        exp, inner = self._expander(_("Appearance"))
        self._append_row(inner, _("Needle color"), self._color_btn(data.NEEDLE_COLOR, lambda c: [setattr(data, "NEEDLE_COLOR", c), refresh()], "255,0,0"), expand=False)
        self._optional_color_row(inner, "Background", data, "BACKGROUND_COLOR", "26,26,26", refresh)
        self._content.append(exp)

        # Center text
        exp, inner = self._expander(_("Center text"), expanded=False)
        inner.append(self._toggle(bool(data.SHOW_TEXT), _("Show text"), lambda v: [setattr(data, "SHOW_TEXT", v), refresh()]))
        inner.append(self._toggle(bool(data.SHOW_UNIT), _("Show unit"), lambda v: [setattr(data, "SHOW_UNIT", v), refresh()]))
        self._font_row(inner, data, refresh)
        self._append_row(inner, _("Font color"), self._color_btn(data.FONT_COLOR, lambda c: [setattr(data, "FONT_COLOR", c), refresh()], "255,255,255"), expand=False)
        self._content.append(exp)

    # ── StatusBar ──────────────────────────────────────────────────────────────

    def _build_statusbar_props(self, elem):
        data = elem.data

        def refresh():
            if not self._updating:
                elem.invalidate()

        def resize():
            elem.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 20)
            refresh()

        # Position & Size
        exp, inner = self._expander(_("Position & Size"))
        self._x_spin = self._spin(data.X, 0, 9999, lambda s: [setattr(data, "X", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._y_spin = self._spin(data.Y, 0, 9999, lambda s: [setattr(data, "Y", int(s.get_value())), elem.set_position(data.X, data.Y), refresh()])
        self._append_row(inner, _("X"), self._x_spin)
        self._append_row(inner, _("Y"), self._y_spin)
        self._append_row(inner, _("Width"),  self._spin(data.WIDTH  or 200, 10, 9999, lambda s: [setattr(data, "WIDTH",  int(s.get_value())), resize()]))
        self._append_row(inner, _("Height"), self._spin(data.HEIGHT or 20,  4,  500,  lambda s: [setattr(data, "HEIGHT", int(s.get_value())), resize()]))
        self._content.append(exp)

        # Range
        exp, inner = self._expander(_("Range"))
        self._append_row(inner, _("Min"), self._spin(data.MIN_VALUE or 0,   -9999, 9999, lambda s: [setattr(data, "MIN_VALUE", int(s.get_value())), refresh()]))
        self._append_row(inner, _("Max"), self._spin(data.MAX_VALUE or 100, -9999, 9999, lambda s: [setattr(data, "MAX_VALUE", int(s.get_value())), refresh()]))
        self._content.append(exp)

        # Appearance
        exp, inner = self._expander(_("Appearance"))
        self._append_row(inner, _("Bar color"),       self._color_btn(data.BAR_COLOR,       lambda c: [setattr(data, "BAR_COLOR",       c), refresh()], "0,255,0"),   expand=False)
        self._append_row(inner, _("Indicator color"), self._color_btn(data.INDICATOR_COLOR, lambda c: [setattr(data, "INDICATOR_COLOR", c), refresh()], "255,255,255"), expand=False)
        self._append_row(inner, _("Indicator radius"), self._spin(data.INDICATOR_RADIUS or 0, 0, 50, lambda s: [setattr(data, "INDICATOR_RADIUS", int(s.get_value())), refresh()]))
        self._optional_color_row(inner, "Background", data, "BACKGROUND_COLOR", "26,26,26", refresh)
        self._content.append(exp)
