"""
Theme editor canvas.
Uses GtkFixed for absolute positioning of draggable elements.
Background layers are GtkPicture stacked under the elements.
"""
import os
import logging
import dataclasses
import typing
from typing import Optional

import cairo
from gi.repository import Gtk, Gdk, GdkPixbuf, GLib

from theme.models import (
    Theme, Text, Graph, Radial, Chart, Gauge, StatusBar, StaticImage,
)
from ui.draggable import (
    DraggableText, DraggableGraph, DraggableRadial, DraggableChart,
    DraggableGauge, DraggableStatusBar, DraggableImage, _DraggableBase,
)

_KIND_TO_SUFFIX = {
    "text":         "TEXT",
    "percent_text": "PERCENT_TEXT",
    "graph":        "GRAPH",
    "radial":       "RADIAL",
    "chart":        "CHART",
    "gauge":        "GAUGE",
    "status_bar":   "STATUS_BAR",
}
# Maps an element kind to the attribute name it occupies on its parent
# Measurement (e.g. "percent_text" → m.PercentText). Used by swap_element_type
# to move data between slots so a type change survives save.
_KIND_TO_SLOT = {
    "text":         "Text",
    "percent_text": "PercentText",
    "graph":        "Graph",
    "radial":       "Radial",
    "chart":        "Chart",
    "gauge":        "Gauge",
    "status_bar":   "StatusBar",
}
_KNOWN_SUFFIXES = {"TEXT", "PERCENT_TEXT", "GRAPH", "RADIAL", "CHART", "GAUGE", "STATUS_BAR"}


def _replace_suffix(path: str, new_suffix: str) -> str:
    """Swap the trailing type suffix of a yaml_path (e.g. ...PERCENT_TEXT → ...TEXT).
    Appends new_suffix if the path has no known suffix."""
    parts = path.rsplit(".", 1)
    if len(parts) == 2 and parts[1] in _KNOWN_SUFFIXES:
        return parts[0] + "." + new_suffix
    return path + "." + new_suffix

log = logging.getLogger(__name__)

# yaml_path components whose dataclass attribute name doesn't match
# case-insensitively. NET paths use WLO/ETH but the attrs are Wifi/Wired.
_PATH_ALIASES = {"WLO": "Wifi", "ETH": "Wired"}


def _norm(s: str) -> str:
    """Normalise a key for comparison: lowercase, drop underscores/dashes.
    YAML uses underscores (PERCENT_TEXT) while the Python attrs don't
    (PercentText), so a plain case-fold won't match."""
    return s.lower().replace("_", "").replace("-", "")


def _ci_field(obj, name: str):
    """Dataclass field on obj matching `name` case-insensitively, ignoring
    underscores/dashes (honouring NET path aliases). Returns the attr or None."""
    if not dataclasses.is_dataclass(obj):
        return None
    target = _norm(_PATH_ALIASES.get(name.upper(), name))
    for f in dataclasses.fields(obj):
        if _norm(f.name) == target:
            return f.name
    return None


def _resolved_type(obj, attr: str):
    """Concrete class declared for `attr` on obj's dataclass (Optional unwrapped)."""
    hints = typing.get_type_hints(type(obj))
    tp = hints.get(attr)
    if tp is None:
        return None
    if typing.get_origin(tp) is typing.Union:
        tp = next((a for a in typing.get_args(tp) if a is not type(None)), None)
    return tp if isinstance(tp, type) else None


def _has_slot(meas, slot_attr: str) -> bool:
    """True if `slot_attr` is a declared dataclass field on the measurement
    (so the representation is valid for this STATS type). Checks declared
    fields only — a dynamically-set attribute doesn't count."""
    return (meas is not None and dataclasses.is_dataclass(meas)
            and any(f.name == slot_attr for f in dataclasses.fields(meas)))

SCREEN_TURZX_W = 1280
SCREEN_TURZX_H = 720
SCREEN_REVC_W  = 800
SCREEN_REVC_H  = 480


class ThemeCanvas(Gtk.ScrolledWindow):
    """
    Scrollable container holding the canvas overlay:
      GtkOverlay
        └── GtkFixed (canvas_fixed)       ← background images + draggable elements
    """

    def __init__(self, properties_panel, layers_panel):
        super().__init__()
        self.set_hexpand(True)
        self.set_vexpand(True)
        self.set_policy(Gtk.PolicyType.AUTOMATIC, Gtk.PolicyType.AUTOMATIC)

        self._props = properties_panel
        self._layers = layers_panel
        self._theme: Optional[Theme] = None
        self._theme_dir: str = ""
        self._canvas_w = SCREEN_TURZX_W
        self._canvas_h = SCREEN_TURZX_H

        self._elements: list[_DraggableBase] = []
        self._selected: Optional[_DraggableBase] = None
        # Background images: key → {"pic", "img", "texture"}
        self._bg_imgs: dict[str, dict] = {}
        self._selected_bg: Optional[str] = None
        self._video_player = None

        self._build_canvas()

    # ------------------------------------------------------------------
    # Construction
    # ------------------------------------------------------------------

    def _build_canvas(self):
        # Outer fixed: sets the overall canvas size so scrollbars work
        self._outer = Gtk.Fixed()
        self._outer.set_size_request(self._canvas_w, self._canvas_h)

        # Dark background rectangle (drawn by CSS / color widget)
        bg_rect = Gtk.DrawingArea()
        bg_rect.set_size_request(self._canvas_w, self._canvas_h)
        bg_rect.set_draw_func(self._draw_bg)
        self._outer.put(bg_rect, 0, 0)
        self._bg_draw = bg_rect

        # GtkFixed for actual content (backgrounds + elements)
        self._fixed = Gtk.Fixed()
        self._fixed.set_size_request(self._canvas_w, self._canvas_h)
        self._outer.put(self._fixed, 0, 0)

        self.set_child(self._outer)

    def _draw_bg(self, area, cr, w, h):
        cr.set_source_rgb(0.08, 0.08, 0.08)
        cr.paint()

    # ------------------------------------------------------------------
    # Theme loading
    # ------------------------------------------------------------------

    def load_theme(self, theme: Theme, theme_dir: str):
        self._theme = theme
        self._theme_dir = theme_dir
        if self._props and hasattr(self._props, "set_theme_dir"):
            self._props.set_theme_dir(theme_dir)
        self._clear_all()

        # Canvas size from theme display
        if theme.display:
            w = theme.display.WIDTH or SCREEN_TURZX_W
            h = theme.display.HEIGHT or SCREEN_TURZX_H
            self._set_canvas_size(w, h)

        # Background layers
        if theme.static_images:
            for key in sorted(theme.static_images.keys()):
                self._load_bg_image(key, theme.static_images[key])

        # Static texts
        if theme.static_texts:
            for path, txt_data in theme.static_texts.items():
                self._add_text_element(txt_data, path)

        # STATS elements
        if theme.STATS:
            self._load_stats(theme.STATS)

        # Bind _measurement/_slot on all loaded elements so the Properties
        # panel can filter the Type dropdown correctly (newly-added elements
        # get these set via _attach_to_theme; loaded ones need it done here).
        self._bind_all_measurements()

        # Sort elements by INDEX for correct z-order
        self._apply_index_order()

        self._layers.refresh(theme, self._elements)

    def _apply_index_order(self):
        """Sort canvas elements by their INDEX field and re-stack in GtkFixed."""
        self._elements.sort(key=lambda e: getattr(e.data, "INDEX", 0))
        for elem in self._elements:
            self._fixed.remove(elem.widget)
            self._fixed.put(elem.widget, elem._x, elem._y)

    def render_to_png(self, path: str):
        """Composite the current canvas (background + all elements in z-order)
        to a PNG. Used to regenerate the theme's home-screen preview on save."""
        w, h = self._canvas_w, self._canvas_h
        surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, w, h)
        cr = cairo.Context(surface)

        # Backdrop (BACKGROUND image), if any.
        info = self._bg_imgs.get("BACKGROUND")
        if info and info.get("pixbuf") is not None:
            pb = info["pixbuf"]
            img = info["img"]
            iw = img.WIDTH or w
            ih = img.HEIGHT or h
            cr.save()
            cr.translate(img.X, img.Y)
            cr.scale(iw / max(1, pb.get_width()), ih / max(1, pb.get_height()))
            Gdk.cairo_set_source_pixbuf(cr, pb, 0, 0)
            cr.paint()
            cr.restore()

        # Elements in z-order (already sorted). Suppress selection highlights so
        # the preview is clean, without disturbing live selection state.
        for elem in self._elements:
            # get_allocated_width() is 0 before the widget is laid out; use the
            # preferred (requested) size so off-screen renders produce correct output.
            min_req, _ = elem.widget.get_preferred_size()
            ew = min_req.width or elem.widget.get_allocated_width() or 1
            eh = min_req.height or elem.widget.get_allocated_height() or 1
            was_sel = getattr(elem, "_selected", False)
            elem._selected = False
            cr.save()
            cr.translate(elem._x, elem._y)
            try:
                elem._draw(None, cr, ew, eh)
            except Exception:
                log.exception("render_to_png: falha ao desenhar %s",
                              getattr(elem, "yaml_path", "?"))
            cr.restore()
            elem._selected = was_sel

        surface.write_to_png(path)

    def capture_z_order(self):
        """Write current stack position as INDEX on each element's data.

        Starts at 1, not 0: INDEX 0 is reserved for the background base layer,
        so sensor widgets/overlays beginning at 0 collide with it and render
        wrong.  1-based matches the hand-authored themes.
        """
        for i, elem in enumerate(self._elements):
            if hasattr(elem.data, "INDEX"):
                elem.data.INDEX = i + 1

    def _bind_all_measurements(self):
        """Set _measurement/_slot on loaded elements that don't have them yet.

        Elements loaded from theme.yaml via _load_stats are created without a
        parent measurement reference; this post-pass navigates the in-memory
        theme to fill it in, so the Properties panel can filter the Type
        dropdown to only the representations that exist on each measurement."""
        if not self._theme:
            return
        for elem in self._elements:
            if getattr(elem, "_measurement", None) is not None:
                continue  # already set (newly added via add_new_element)
            path = getattr(elem, "yaml_path", "")
            if not path or "." not in path:
                continue
            parts = path.split(".")
            # Only resolve elements whose terminal segment is a known type suffix
            # (TEXT, GRAPH, …). Non-suffix terminals (USED, FREE, CONDITION…)
            # live in a text-only slot and intentionally keep _measurement=None.
            if parts[-1] not in _KNOWN_SUFFIXES:
                continue
            obj = self._theme
            for part in parts[:-1]:
                attr = _ci_field(obj, part)
                if attr is None:
                    obj = None
                    break
                obj = getattr(obj, attr, None)
                if obj is None:
                    break
            if obj is None or not dataclasses.is_dataclass(obj):
                continue
            slot_attr = _ci_field(obj, parts[-1])
            if slot_attr:
                elem._measurement = obj
                elem._slot = slot_attr

    def _load_stats(self, stats):
        if stats.CPU:
            cpu = stats.CPU
            for attr, base in [
                ("Percentage", "STATS.CPU.PERCENTAGE"),
                ("Temperature", "STATS.CPU.TEMPERATURE"),
                ("Frequency",  "STATS.CPU.FREQUENCY"),
                ("Fan",        "STATS.CPU.FAN"),
                ("Power",      "STATS.CPU.POWER"),
                ("Voltage",    "STATS.CPU.VOLTAGE"),
            ]:
                m = getattr(cpu, attr, None)
                if m:
                    self._load_measurement(m, base)
            if cpu.Load:
                for sub, path in [
                    (cpu.Load.One,     "STATS.CPU.LOAD.ONE.TEXT"),
                    (cpu.Load.Five,    "STATS.CPU.LOAD.FIVE.TEXT"),
                    (cpu.Load.Fifteen, "STATS.CPU.LOAD.FIFTEEN.TEXT"),
                ]:
                    if sub and sub.Text:
                        self._add_text_element(sub.Text, path)

        if stats.GPU:
            for attr, base in [
                ("Percentage",  "STATS.GPU.PERCENTAGE"),
                ("Temperature", "STATS.GPU.TEMPERATURE"),
                ("Memory",      "STATS.GPU.MEMORY"),
                ("Power",       "STATS.GPU.POWER"),
                ("Frequency",   "STATS.GPU.FREQUENCY"),
                ("Fan",         "STATS.GPU.FAN"),
            ]:
                m = getattr(stats.GPU, attr, None)
                if m:
                    self._load_measurement(m, base)

        if stats.Memory:
            for sub, base in [
                (stats.Memory.Virtual, "STATS.MEMORY.VIRTUAL"),
                (stats.Memory.Swap,    "STATS.MEMORY.SWAP"),
            ]:
                if sub:
                    if sub.PercentText:
                        self._add_text_element(sub.PercentText, base + ".PERCENT_TEXT")
                    if sub.Used:
                        self._add_text_element(sub.Used, base + ".USED")
                    if sub.Free:
                        self._add_text_element(sub.Free, base + ".FREE")
                    if sub.Graph:
                        self._add_graph_element(sub.Graph, base + ".GRAPH")
                    if sub.Radial:
                        self._add_radial_element(sub.Radial, base + ".RADIAL")
                    if sub.Chart:
                        self._add_chart_element(sub.Chart, base + ".CHART")

        if stats.Disk:
            for attr, base in [
                ("Used",  "STATS.DISK.USED"),
                ("Free",  "STATS.DISK.FREE"),
                ("Total", "STATS.DISK.TOTAL"),
                ("Temperature", "STATS.DISK.TEMPERATURE"),
            ]:
                m = getattr(stats.Disk, attr, None)
                if m:
                    self._load_measurement(m, base)

        if stats.Net:
            for iface, iface_key in [(stats.Net.Wifi, "WLO"), (stats.Net.Wired, "ETH")]:
                if iface:
                    base = f"STATS.NET.{iface_key}"
                    for attr, sub in [("Upload","UPLOAD"),("Download","DOWNLOAD"),
                                       ("Uploaded","UPLOADED"),("Downloaded","DOWNLOADED")]:
                        m = getattr(iface, attr, None)
                        if m:
                            self._load_measurement(m, f"{base}.{sub}")

        if stats.Date:
            for attr, base in [("Day","STATS.DATE.DAY"), ("Hour","STATS.DATE.HOUR")]:
                m = getattr(stats.Date, attr, None)
                if m:
                    self._load_measurement(m, base)

        if stats.Weather:
            if stats.Weather.Temperature:
                self._load_measurement(stats.Weather.Temperature, "STATS.WEATHER.TEMPERATURE")
            if stats.Weather.Condition:
                self._add_text_element(stats.Weather.Condition, "STATS.WEATHER.CONDITION")

        if stats.Volume and stats.Volume.Text:
            self._add_text_element(stats.Volume.Text, "STATS.VOLUME.TEXT")

    def _load_measurement(self, m, base: str):
        if m.Text:
            self._add_text_element(m.Text, base + ".TEXT", m, "Text")
        if m.PercentText:
            self._add_text_element(m.PercentText, base + ".PERCENT_TEXT", m, "PercentText")
        if m.Graph:
            self._add_graph_element(m.Graph, base + ".GRAPH", m, "Graph")
        if m.Radial:
            self._add_radial_element(m.Radial, base + ".RADIAL", m, "Radial")
        if m.Chart:
            self._add_chart_element(m.Chart, base + ".CHART", m, "Chart")
        if hasattr(m, "Gauge") and m.Gauge:
            self._add_gauge_element(m.Gauge, base + ".GAUGE", m, "Gauge")
        if hasattr(m, "StatusBar") and m.StatusBar:
            self._add_statusbar_element(m.StatusBar, base + ".STATUS_BAR", m, "StatusBar")

    # ------------------------------------------------------------------
    # Background images
    # ------------------------------------------------------------------

    def _load_bg_image(self, key: str, img_data: StaticImage):
        path = os.path.join(self._theme_dir, img_data.PATH)
        if not os.path.exists(path):
            log.warning("Background image not found: %s", path)
            return
        if key == "BACKGROUND":
            # Pinned backdrop — always behind everything. Kept as a Picture.
            try:
                scaled = GdkPixbuf.Pixbuf.new_from_file_at_scale(
                    path, img_data.WIDTH or self._canvas_w,
                    img_data.HEIGHT or self._canvas_h, False)
                texture = Gdk.Texture.new_for_pixbuf(scaled)
                pic = Gtk.Picture.new_for_paintable(texture)
                pic.set_size_request(img_data.WIDTH or self._canvas_w,
                                      img_data.HEIGHT or self._canvas_h)
                pic.set_content_fit(Gtk.ContentFit.FILL)
                self._fixed.put(pic, img_data.X, img_data.Y)
                click = Gtk.GestureClick()
                click.connect("pressed", lambda *_: self._select_bg(key))
                pic.add_controller(click)
                self._bg_imgs[key] = {"pic": pic, "img": img_data,
                                      "texture": texture, "pixbuf": scaled}
            except Exception as e:
                log.error("Failed to load background %s: %s", path, e)
            return
        # Non-BACKGROUND image → z-ordered element (sits above/below widgets by
        # INDEX), matching how the daemon renders it.
        try:
            pixbuf = GdkPixbuf.Pixbuf.new_from_file(path)
            elem = DraggableImage(img_data, pixbuf, f"static_images.{key}",
                                  self._canvas_w, self._canvas_h)
            self._add_element(elem)
        except Exception as e:
            log.error("Failed to load image %s: %s", path, e)

    def _apply_bg_geometry(self, key: str):
        """Move/resize a background picture to match its StaticImage X/Y/W/H."""
        info = self._bg_imgs.get(key)
        if not info:
            return
        img = info["img"]
        pic = info["pic"]
        w = img.WIDTH or self._canvas_w
        h = img.HEIGHT or self._canvas_h
        pic.set_size_request(w, h)
        self._fixed.move(pic, img.X, img.Y)

    def _select_bg(self, key: str):
        # Selecting a background image clears any selected element.
        self.select_element(None)
        self._selected_bg = key
        img = self._bg_imgs.get(key, {}).get("img")
        if img is not None and self._props is not None:
            self._props.update_bg_image(key, img, lambda: self._apply_bg_geometry(key))
        if self._layers is not None:
            self._layers.select_bg(key)

    def select_bg_image(self, key: str):
        """Public entry for the layers panel to select a background image."""
        if key in self._bg_imgs:
            self._select_bg(key)

    def add_background_layer(self, path: str):
        if not self._theme:
            return
        if self._theme.static_images is None:
            self._theme.static_images = {}
        key = self._gen_layer_key()
        import PIL.Image  # optional — fallback to just storing path
        try:
            from gi.repository import GdkPixbuf as _PB
            pb = _PB.Pixbuf.new_from_file(path)
            w, h = pb.get_width(), pb.get_height()
        except Exception:
            w, h = self._canvas_w, self._canvas_h
        self._theme.static_images[key] = StaticImage(
            PATH=os.path.basename(path), X=0, Y=0, WIDTH=w, HEIGHT=h, INDEX=9999
        )
        self.load_theme(self._theme, os.path.dirname(path) or self._theme_dir)

    def remove_background_layer(self, key: str):
        if self._theme and key in (self._theme.static_images or {}):
            del self._theme.static_images[key]
            self.load_theme(self._theme, self._theme_dir)

    def _gen_layer_key(self) -> str:
        existing = set((self._theme.static_images or {}).keys())
        if "BACKGROUND" not in existing:
            return "BACKGROUND"
        for i in range(2, 100):
            k = f"LAYER_{i}"
            if k not in existing:
                return k
        return f"LAYER_{len(existing)+1}"

    # ------------------------------------------------------------------
    # Element management
    # ------------------------------------------------------------------

    def _add_element(self, elem: _DraggableBase):
        elem._fixed = self._fixed
        elem.set_on_selected(self._on_element_selected)
        elem.set_on_drag_end(self._on_element_drag_end)
        self._fixed.put(elem.widget, elem._x, elem._y)
        self._elements.append(elem)

    def _resolve_slot(self, path: str):
        """Navigate self._theme by a yaml_path, creating missing intermediate
        objects from their declared field types. Returns (parent, slot_attr)
        for the leaf, or (None, None)."""
        obj = self._theme
        if obj is None:
            return None, None
        parts = path.split(".")
        for part in parts[:-1]:
            attr = _ci_field(obj, part)
            if attr is None:
                return None, None
            nxt = getattr(obj, attr, None)
            if nxt is None:
                tp = _resolved_type(obj, attr)
                if tp is None:
                    return None, None
                nxt = tp()
                setattr(obj, attr, nxt)
            obj = nxt
        leaf = _ci_field(obj, parts[-1])
        if leaf is None:
            return None, None
        return obj, leaf

    def _slot_valid(self, path: str) -> bool:
        """True if the path's leaf slot exists on its measurement's dataclass.
        Navigates the existing tree; for a not-yet-created measurement it
        instantiates the declared type transiently (never stored) just to check
        the slot. This is the general, data-driven validity check for any STATS
        measurement type — no per-type hardcoding."""
        if not self._theme or not path:
            return False
        obj = self._theme
        parts = path.split(".")
        for part in parts[:-1]:
            attr = _ci_field(obj, part)
            if attr is None:
                return False
            nxt = getattr(obj, attr, None)
            if nxt is None:
                tp = _resolved_type(obj, attr)
                if tp is None:
                    return False
                obj = tp()  # transient instance, only for type-checking
            else:
                obj = nxt
        return _ci_field(obj, parts[-1]) is not None

    def _attach_to_theme(self, elem: _DraggableBase):
        """Attach a freshly-created element's data into self._theme at its
        yaml_path. The in-memory theme is the single source of truth that save
        serializes, so without this a new element lives only on the canvas and
        is dropped on save."""
        path = getattr(elem, "yaml_path", "")
        if not self._theme or not path:
            return
        parent, slot = self._resolve_slot(path)
        if parent is None:
            log.warning("Não foi possível anexar o elemento ao tema: %s", path)
            return
        setattr(parent, slot, elem.data)
        elem._measurement = parent
        elem._slot = slot

    def _add_text_element(self, data: Text, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableText(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar texto '%s'", path)

    def _add_graph_element(self, data: Graph, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableGraph(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar graph '%s'", path)

    def _add_radial_element(self, data: Radial, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableRadial(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar radial '%s'", path)

    def _add_chart_element(self, data: Chart, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableChart(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar chart '%s'", path)

    def _add_gauge_element(self, data: Gauge, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableGauge(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar gauge '%s'", path)

    def _add_statusbar_element(self, data: StatusBar, path: str, measurement=None, slot=None):
        if data.SHOW is False:
            return
        try:
            elem = DraggableStatusBar(data, path, self._canvas_w, self._canvas_h)
            elem._measurement, elem._slot = measurement, slot
            self._add_element(elem)
        except Exception:
            log.exception("Falha ao criar statusbar '%s'", path)

    def _displace_slot(self, meas, slot, keep):
        """If another element already occupies the (meas, slot) pair — and it
        isn't `keep` — remove it, so a type switch never leaves two elements
        writing the same measurement slot (e.g. converting PercentText→Text when
        a Text already exists)."""
        if meas is None or slot is None:
            return
        victim = None
        for e in self._elements:
            if e is keep:
                continue
            if getattr(e, "_measurement", None) is meas and getattr(e, "_slot", None) == slot:
                victim = e
                break
        if victim is not None:
            self._fixed.remove(victim.widget)
            self._elements.remove(victim)
            if self._selected is victim:
                self._selected = keep

    def swap_element_type(self, elem: _DraggableBase, new_type: str):
        """Replace elem with a new element of new_type.

        For text <-> percent_text the model is the same (Text), so we move the
        data object between the measurement's m.Text / m.PercentText slots and
        update the yaml_path suffix — every field is preserved and the change
        survives save. Other transitions rebuild from shared fields and reattach
        the new model to the measurement slot."""
        if elem not in self._elements:
            return
        old = elem.data
        x, y = elem.get_model_position()
        meas = getattr(elem, "_measurement", None)
        old_slot = getattr(elem, "_slot", None)

        # Refuse a swap whose target slot doesn't exist on this measurement
        # type (e.g. Memory → 'Text': MemMeasurement has no Text slot). Checked
        # against the real declared fields, so it's correct for every STATS.
        new_slot = _KIND_TO_SLOT.get(new_type)
        if meas is not None and new_slot and not _has_slot(meas, new_slot):
            log.warning("Troca para '%s' inválida para este sensor", new_type)
            return

        # text <-> percent_text: same Text model, just move the slot.
        if (isinstance(elem, DraggableText) and new_type in ("text", "percent_text")
                and old_slot in ("Text", "PercentText")):
            new_slot = _KIND_TO_SLOT[new_type]
            if new_slot == old_slot:
                return
            self._displace_slot(meas, new_slot, elem)
            elem.yaml_path = _replace_suffix(elem.yaml_path, _KIND_TO_SUFFIX[new_type])
            if meas is not None:
                setattr(meas, old_slot, None)
                setattr(meas, new_slot, old)
            elem._slot = new_slot
            self._on_element_selected(elem)
            self._layers.refresh(self._theme, self._elements)
            return

        # Shared field extraction helpers
        def _get(attr, default=None):
            return getattr(old, attr, default)

        index   = _get("INDEX", 0)
        show    = _get("SHOW", True)
        min_val = _get("MIN_VALUE", 0)
        max_val = _get("MAX_VALUE", 100)
        width   = _get("WIDTH", 200)
        height  = _get("HEIGHT", 100)
        bg      = _get("BACKGROUND_COLOR", "")
        grad    = _get("GRADIENT_COLOR", "")
        bar     = (_get("BAR_COLOR") or _get("NEEDLE_COLOR") or
                   _get("FILL_COLOR") or _get("FONT_COLOR") or "0,255,0")

        path = _replace_suffix(elem.yaml_path, _KIND_TO_SUFFIX[new_type]) \
            if new_type in _KIND_TO_SUFFIX else elem.yaml_path

        if new_type == "text":
            data = Text(X=x, Y=y, SHOW=show, INDEX=index,
                        FONT_COLOR=bar, BACKGROUND_COLOR=bg)
            new_elem = DraggableText(data, path, self._canvas_w, self._canvas_h)
        elif new_type == "graph":
            data = Graph(X=x, Y=y, SHOW=show, INDEX=index,
                         WIDTH=width, HEIGHT=height,
                         MIN_VALUE=min_val, MAX_VALUE=max_val,
                         BAR_COLOR=bar, BACKGROUND_COLOR=bg, GRADIENT_COLOR=grad)
            new_elem = DraggableGraph(data, path, self._canvas_w, self._canvas_h)
        elif new_type == "radial":
            data = Radial(X=x, Y=y, SHOW=show, INDEX=index,
                          MIN_VALUE=min_val, MAX_VALUE=max_val,
                          BAR_COLOR=bar, BACKGROUND_COLOR=bg, GRADIENT_COLOR=grad)
            new_elem = DraggableRadial(data, path, self._canvas_w, self._canvas_h)
        elif new_type == "chart":
            data = Chart(X=x, Y=y, SHOW=show, INDEX=index,
                         WIDTH=width, HEIGHT=height,
                         MIN_VALUE=min_val, MAX_VALUE=max_val,
                         FILL_COLOR=bar)
            new_elem = DraggableChart(data, path, self._canvas_w, self._canvas_h)
        elif new_type == "gauge":
            data = Gauge(X=x, Y=y, SHOW=show, INDEX=index,
                         MIN_VALUE=min_val, MAX_VALUE=max_val,
                         NEEDLE_COLOR=bar, BACKGROUND_COLOR=bg)
            new_elem = DraggableGauge(data, path, self._canvas_w, self._canvas_h)
        elif new_type == "status_bar":
            data = StatusBar(X=x, Y=y, SHOW=show, INDEX=index,
                             WIDTH=width, HEIGHT=height,
                             MIN_VALUE=min_val, MAX_VALUE=max_val,
                             BAR_COLOR=bar, BACKGROUND_COLOR=bg)
            new_elem = DraggableStatusBar(data, path, self._canvas_w, self._canvas_h)
        else:
            return

        # Reattach to the measurement so the swap survives save: drop the old
        # slot and put the new model in the slot matching new_type. Displace any
        # pre-existing element already occupying that slot first.
        new_slot = _KIND_TO_SLOT.get(new_type)
        self._displace_slot(meas, new_slot, elem)
        if meas is not None:
            if old_slot:
                setattr(meas, old_slot, None)
            if new_slot:
                setattr(meas, new_slot, data)
        new_elem._measurement = meas
        new_elem._slot = new_slot

        # Remove old, insert new at same z-order position. (Recompute stack_idx
        # here — _displace_slot above may have removed an earlier element.)
        stack_idx = self._elements.index(elem)
        self._fixed.remove(elem.widget)
        self._elements.remove(elem)
        if self._selected is elem:
            self._selected = None

        new_elem._fixed = self._fixed
        new_elem.set_on_selected(self._on_element_selected)
        new_elem.set_on_drag_end(self._on_element_drag_end)
        self._elements.insert(stack_idx, new_elem)

        # Re-stack all elements to restore correct z-order
        for e in self._elements:
            self._fixed.remove(e.widget)
            self._fixed.put(e.widget, e._x, e._y)

        self._on_element_selected(new_elem)
        self._layers.refresh(self._theme, self._elements)

    def remove_element(self, elem: _DraggableBase):
        if elem in self._elements:
            # Drop from the in-memory theme so the deletion survives save/reload.
            if isinstance(elem, DraggableImage) and elem.yaml_path.startswith("static_images."):
                key = elem.yaml_path.split(".", 1)[1]
                if self._theme and key in (self._theme.static_images or {}):
                    del self._theme.static_images[key]
            else:
                # STATS widget: clear its slot on the parent measurement so the
                # removed element doesn't reappear on the next save.
                meas = getattr(elem, "_measurement", None)
                slot = getattr(elem, "_slot", None)
                if meas is not None and slot and getattr(meas, slot, None) is elem.data:
                    setattr(meas, slot, None)
            self._fixed.remove(elem.widget)
            self._elements.remove(elem)
            if self._selected is elem:
                self._selected = None
                self._props.update(None)
            self._layers.refresh(self._theme, self._elements)

    def _sync_to_theme(self):
        """Re-attach every canvas element to self._theme. The in-memory theme is
        the single source of truth that save serializes, so this guarantees the
        written YAML matches exactly what's on the canvas — no drift from adds,
        removes, or edits. Idempotent for already-attached elements."""
        if not self._theme:
            return
        for elem in self._elements:
            if isinstance(elem, DraggableImage):
                continue  # static images are tracked in theme.static_images
            self._attach_to_theme(elem)

    def add_new_element(self, yaml_path: str, kind: str):
        """Called from toolbox when user clicks an add button."""
        # Refuse representations that have no slot on this measurement type
        # (e.g. 'Text' on a Memory measurement, which only has USED/FREE/
        # PERCENT_TEXT). Validity is checked against the real dataclass fields,
        # so it's correct for every STATS type without hardcoding.
        if not self._slot_valid(yaml_path):
            log.warning("Representação '%s' inválida para este sensor: %s",
                        kind, yaml_path)
            return
        cx, cy = self._canvas_w // 2, self._canvas_h // 2
        if kind == "text":
            data = Text(X=cx, Y=cy, SHOW=True, FONT_SIZE=24, FONT_COLOR="255,255,255")
            elem = DraggableText(data, yaml_path, self._canvas_w, self._canvas_h)
        elif kind == "graph":
            data = Graph(X=cx, Y=cy, WIDTH=200, HEIGHT=80, SHOW=True, BAR_COLOR="0,255,0")
            elem = DraggableGraph(data, yaml_path, self._canvas_w, self._canvas_h)
        elif kind == "radial":
            data = Radial(X=cx, Y=cy, RADIUS=80, WIDTH=10, SHOW=True, BAR_COLOR="0,255,0")
            elem = DraggableRadial(data, yaml_path, self._canvas_w, self._canvas_h)
        elif kind == "chart":
            data = Chart(X=cx, Y=cy, WIDTH=200, HEIGHT=100, SHOW=True, FILL_COLOR="0,255,0")
            elem = DraggableChart(data, yaml_path, self._canvas_w, self._canvas_h)
        elif kind == "gauge":
            data = Gauge(X=cx, Y=cy, RADIUS=80, SHOW=True, NEEDLE_COLOR="255,0,0")
            elem = DraggableGauge(data, yaml_path, self._canvas_w, self._canvas_h)
        elif kind == "status_bar":
            data = StatusBar(X=cx, Y=cy, WIDTH=200, HEIGHT=20, SHOW=True, BAR_COLOR="0,255,0")
            elem = DraggableStatusBar(data, yaml_path, self._canvas_w, self._canvas_h)
        else:
            return
        self._add_element(elem)
        self._attach_to_theme(elem)
        self._layers.refresh(self._theme, self._elements)
        self._on_element_selected(elem)

    def select_element(self, elem: Optional[_DraggableBase]):
        if self._selected and self._selected is not elem:
            self._selected.set_selected(False)
        self._selected = elem
        if elem:
            # Selecting an element clears any selected background image.
            self._selected_bg = None
            elem.set_selected(True)
        self._props.update(elem)

    def select_widget_by_path(self, path: str):
        """Find and select the element with the given yaml_path."""
        for elem in self._elements:
            if elem.yaml_path == path:
                self._on_element_selected(elem)
                return

    def show_layer_properties(self, key: str, path: str, on_change):
        """Route a layer click to the properties panel."""
        if hasattr(self._props, "show_layer_info"):
            self._props.show_layer_info(key, path, on_change)

    def reorder_element(self, elem: _DraggableBase, direction: int):
        """Move elem one step up/down in the z-order stack and rewrite INDEX.
        Operates on the element reference directly (the layers-panel list also
        contains video/BACKGROUND entries, so list indices don't map 1:1 to the
        widget-only _elements list)."""
        try:
            i = self._elements.index(elem)
        except ValueError:
            return
        j = i + direction
        if not (0 <= j < len(self._elements)):
            return
        self._elements[i], self._elements[j] = self._elements[j], self._elements[i]
        # Re-stack all elements in the new order
        for e in self._elements:
            self._fixed.remove(e.widget)
            self._fixed.put(e.widget, e._x, e._y)
        self.capture_z_order()
        if self._layers and self._theme:
            self._layers.refresh(self._theme, self._elements)
            self._layers.select_element(elem)  # keep the moved element highlighted

    # ------------------------------------------------------------------
    # Event callbacks
    # ------------------------------------------------------------------

    def _on_element_selected(self, elem: _DraggableBase):
        self.select_element(elem)
        self._layers.select_element(elem)

    def _on_element_drag_end(self, elem: _DraggableBase):
        self._props.update_position(elem)

    # ------------------------------------------------------------------
    # Canvas size
    # ------------------------------------------------------------------

    def _set_canvas_size(self, w: int, h: int):
        self._canvas_w = w
        self._canvas_h = h
        self._outer.set_size_request(w, h)
        self._fixed.set_size_request(w, h)
        self._bg_draw.set_size_request(w, h)
        for elem in self._elements:
            elem.set_canvas_size(w, h)

    def _clear_all(self):
        self.select_element(None)
        for elem in self._elements:
            self._fixed.remove(elem.widget)
        self._elements.clear()
        for info in self._bg_imgs.values():
            self._fixed.remove(info["pic"])
        self._bg_imgs.clear()
        self._selected_bg = None
