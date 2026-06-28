"""
Draggable canvas elements — GTK4 version.

Each element is a GtkDrawingArea rendered with Cairo/Pango.
Positioned on a GtkFixed container (the canvas).
GtkGestureDrag handles move; GtkGestureClick handles selection.
"""

from __future__ import annotations
import math
import os
from typing import Callable, Optional, TYPE_CHECKING

import cairo
from gi.repository import Gtk, GLib, Pango, PangoCairo

from theme.models import Text, Graph, Radial, Chart, Gauge, StatusBar

if TYPE_CHECKING:
    pass

SELECTION_COLOR = (0.8, 0.8, 0.8, 1.0)
SELECTION_WIDTH = 1.5

# ──────────────────────────────────────────────────────────────────────────────
# helpers
# ──────────────────────────────────────────────────────────────────────────────

def _parse_color(color_str: str, default=(1.0, 1.0, 1.0, 1.0)):
    """
    Parse color in two formats used by the Go theme engine:
      - '#RRGGBB' / '#RRGGBBAA'  (hex)
      - 'R,G,B' / 'R,G,B,A'    (comma-separated 0-255, Go format)
    """
    s = (color_str or "").strip()
    if not s:
        return default
    if s.startswith("#"):
        h = s.lstrip("#")
        try:
            if len(h) == 6:
                r, g, b = int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)
                return r / 255, g / 255, b / 255, 1.0
            if len(h) == 8:
                r, g, b, a = (int(h[i*2:(i+1)*2], 16) for i in range(4))
                return r / 255, g / 255, b / 255, a / 255
        except ValueError:
            pass
    elif "," in s:
        try:
            parts = [int(x.strip()) for x in s.split(",")]
            if len(parts) == 3:
                return parts[0]/255, parts[1]/255, parts[2]/255, 1.0
            if len(parts) == 4:
                return parts[0]/255, parts[1]/255, parts[2]/255, parts[3]/255
        except (ValueError, IndexError):
            pass
    return default


def _clamp(val, lo, hi):
    return max(lo, min(hi, val))


def _rounded_rect(cr, x, y, w, h, r):
    """Draw a rounded rectangle path (Cairo has no built-in)."""
    r = min(r, w / 2, h / 2)
    if r <= 0:
        cr.rectangle(x, y, w, h)
        return
    cr.new_path()
    cr.arc(x + r,     y + r,     r, math.pi,       3 * math.pi / 2)
    cr.arc(x + w - r, y + r,     r, 3 * math.pi / 2, 0)
    cr.arc(x + w - r, y + h - r, r, 0,               math.pi / 2)
    cr.arc(x + r,     y + h - r, r, math.pi / 2,     math.pi)
    cr.close_path()


# ──────────────────────────────────────────────────────────────────────────────
# Base draggable
# ──────────────────────────────────────────────────────────────────────────────

class _DraggableBase:
    """
    Common drag logic. Subclasses embed a GtkDrawingArea as self.widget.
    The parent canvas registers itself via set_canvas().
    """

    def __init__(self, canvas_w: int, canvas_h: int):
        self._canvas_w = canvas_w
        self._canvas_h = canvas_h
        self._selected = False
        self._on_selected: Optional[Callable] = None
        self._on_drag_end: Optional[Callable] = None
        self._fixed: Optional[Gtk.Fixed] = None   # set by canvas on add

        # position in canvas coords
        self._x = 0.0
        self._y = 0.0

        # drag tracking
        self._drag_start_x = 0.0
        self._drag_start_y = 0.0
        # Accumulated widget movement during a drag.
        # GestureDrag reports offsets in widget-local coords. When we move
        # the widget via Gtk.Fixed.move(), the same screen position maps to
        # a different widget-local coord (shifted by -delta). Without
        # compensation the element oscillates between old and new positions.
        self._drag_accum_x = 0.0
        self._drag_accum_y = 0.0

        self.widget = Gtk.DrawingArea()
        self.widget.set_draw_func(self._draw)
        self.widget.set_focusable(True)

        # Drag gesture
        drag = Gtk.GestureDrag()
        drag.connect("drag-begin", self._on_drag_begin)
        drag.connect("drag-update", self._on_drag_update)
        drag.connect("drag-end", self._on_drag_end_cb)
        self.widget.add_controller(drag)

        # Click gesture (selection)
        click = Gtk.GestureClick()
        click.connect("pressed", self._on_click)
        self.widget.add_controller(click)

    def set_position(self, x: int, y: int):
        self._x = float(x)
        self._y = float(y)
        if self._fixed:
            self._fixed.move(self.widget, self._x, self._y)

    def get_position(self) -> tuple[int, int]:
        return int(self._x), int(self._y)

    def get_model_position(self) -> tuple[int, int]:
        """Position in model/data coordinates for the properties panel spins.
        Base: same as widget top-left. Override for elements with offset anchors."""
        return int(self._x), int(self._y)

    def set_canvas_size(self, w: int, h: int):
        self._canvas_w = w
        self._canvas_h = h

    def set_selected(self, selected: bool):
        self._selected = selected
        self.widget.queue_draw()

    def set_on_selected(self, cb: Callable):
        self._on_selected = cb

    def set_on_drag_end(self, cb: Callable):
        self._on_drag_end = cb

    # ── Gestures ──────────────────────────────────────────────────────

    def _on_click(self, gesture, n_press, x, y):
        if self._on_selected:
            self._on_selected(self)

    def _on_drag_begin(self, gesture, start_x, start_y):
        self._drag_start_x = self._x
        self._drag_start_y = self._y
        self._drag_accum_x = 0.0
        self._drag_accum_y = 0.0
        if self._on_selected:
            self._on_selected(self)

    def _on_drag_update(self, gesture, offset_x, offset_y):
        w = self.widget.get_allocated_width()
        h = self.widget.get_allocated_height()

        real_x = offset_x + self._drag_accum_x
        real_y = offset_y + self._drag_accum_y

        new_x = _clamp(self._drag_start_x + real_x, 0, self._canvas_w - w)
        new_y = _clamp(self._drag_start_y + real_y, 0, self._canvas_h - h)

        dx = new_x - self._x
        dy = new_y - self._y
        if self._fixed and (dx or dy):
            self._drag_accum_x += dx
            self._drag_accum_y += dy
            self._fixed.move(self.widget, new_x, new_y)

        self._x = new_x
        self._y = new_y

    def _on_drag_end_cb(self, gesture, offset_x, offset_y):
        self._commit_position()
        if self._on_drag_end:
            self._on_drag_end(self)

    def _commit_position(self):
        """Override in subclass to write back to data model."""
        pass

    def _draw_selection(self, cr: cairo.Context, w: float, h: float):
        if not self._selected:
            return
        cr.set_source_rgba(*SELECTION_COLOR)
        cr.set_line_width(SELECTION_WIDTH)
        cr.set_dash([4, 4])
        cr.rectangle(0.5, 0.5, w - 1, h - 1)
        cr.stroke()
        cr.set_dash([])

    def _draw(self, area, cr, w, h):
        raise NotImplementedError


# ──────────────────────────────────────────────────────────────────────────────
# DraggableText
# ──────────────────────────────────────────────────────────────────────────────

class DraggableText(_DraggableBase):
    _DPI = 96.0  # match Windows/Go rendering

    def __init__(self, data: Text, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        self._preview_text = self._placeholder_for_path(yaml_path)
        tw, th = self._measure_text()
        self.widget.set_size_request(tw, th)
        self.set_position(self._anchor_to_left(data.X, tw), data.Y)

    def _placeholder_for_path(self, path: str) -> str:
        p = path.upper()
        if "PERCENT" in p:          return "75%"
        if "TEMPERATURE" in p:      return "65°C"
        if "FAN" in p:              return "1200 RPM"
        if "POWER" in p:            return "95W"
        if "VOLTAGE" in p:          return "1.25V"
        if "FREQUENCY" in p or "FREQ" in p: return "3.6 GHz"
        if "HOUR" in p:             return "23:59:59"
        if "DAY" in p:              return "Seg, 01 Jan"
        if "WEATHER" in p:          return "25°C"
        if "VOLUME" in p:           return "85%"
        if "UPLOAD" in p or "DOWNLOAD" in p: return "125 MB/s"
        if "MEMORY" in p or "USED" in p or "FREE" in p: return "8.5 GB"
        return "Sample"

    def invalidate(self):
        """Call after data changes to trigger re-render."""
        tw, th = self._measure_text()
        self.widget.set_size_request(tw, th)
        self.set_position(self._anchor_to_left(self.data.X, tw), self.data.Y)
        self.widget.queue_draw()

    # -- helpers -----------------------------------------------------------

    def _anchor_to_left(self, data_x: int, text_w: int) -> int:
        align = (self.data.ALIGN or "LEFT").upper()
        if align == "CENTER":
            return data_x - text_w // 2
        if align == "RIGHT":
            return data_x - text_w
        return data_x

    def _make_font_desc(self) -> Pango.FontDescription:
        fd = Pango.FontDescription()
        fd.set_absolute_size(max(1, self.data.FONT_SIZE) * Pango.SCALE)
        if self.data.FONT:
            stem = os.path.splitext(os.path.basename(self.data.FONT))[0]
            parts = stem.split("-")
            fd.set_family(parts[0])
            if len(parts) > 1:
                lp = parts[1].lower()
                if "bold" in lp:
                    fd.set_weight(Pango.Weight.BOLD)
                if "italic" in lp or "oblique" in lp:
                    fd.set_style(Pango.Style.ITALIC)
        return fd

    def _display_text(self) -> str:
        return self.data.TEXT or self.data.PLACEHOLDER or self._preview_text

    def _measure_text(self) -> tuple[int, int]:
        ctx = PangoCairo.font_map_get_default().create_context()
        PangoCairo.context_set_resolution(ctx, self._DPI)
        layout = Pango.Layout(ctx)
        layout.set_text(self._display_text(), -1)
        layout.set_font_description(self._make_font_desc())
        ink, logical = layout.get_pixel_extents()
        return max(logical.width, 20), max(logical.height, 12)

    # -- draw --------------------------------------------------------------

    def _draw(self, area, cr, w, h):
        bg = _parse_color(self.data.BACKGROUND_COLOR, (0, 0, 0, 0))
        if bg[3] > 0:
            cr.set_source_rgba(*bg)
            cr.paint()

        layout = PangoCairo.create_layout(cr)
        PangoCairo.context_set_resolution(layout.get_context(), self._DPI)
        layout.context_changed()
        layout.set_text(self._display_text(), -1)
        layout.set_font_description(self._make_font_desc())

        ink, logical = layout.get_pixel_extents()
        tw, th = logical.width, logical.height

        if tw > w or th > h:
            def _resize():
                self.widget.set_size_request(max(tw, 20), max(th, 12))
                self.set_position(self._anchor_to_left(self.data.X, max(tw, 20)), self.data.Y)
            GLib.idle_add(_resize)

        fg = _parse_color(self.data.FONT_COLOR, (1, 1, 1, 1))
        cr.set_source_rgba(*fg)
        PangoCairo.show_layout(cr, layout)

        self._draw_selection(cr, tw, th)

    def get_model_position(self) -> tuple[int, int]:
        """Text stores the alignment anchor (left/center/right) in data.X — not
        the widget's top-left. The properties spinners edit data.X directly, so
        report that rather than self._x, otherwise a click (which fires
        drag-end → update_position) would rewrite the spin to the left-edge
        coordinate and make the value jump on every selection."""
        return self.data.X, self.data.Y

    def _commit_position(self):
        align = (self.data.ALIGN or "LEFT").upper()
        w = self.widget.get_allocated_width()
        if align == "CENTER":
            self.data.X = int(self._x + w / 2)
        elif align == "RIGHT":
            self.data.X = int(self._x + w)
        else:
            self.data.X = int(self._x)
        self.data.Y = int(self._y)


# ──────────────────────────────────────────────────────────────────────────────
# DraggableGraph
# ──────────────────────────────────────────────────────────────────────────────

class DraggableGraph(_DraggableBase):

    def __init__(self, data: Graph, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        self.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 100)
        self.set_position(data.X, data.Y)

    def invalidate(self):
        self.widget.queue_draw()

    def _draw(self, area, cr, w, h):
        # Background
        bg = _parse_color(self.data.BACKGROUND_COLOR, (0.1, 0.1, 0.1, 1.0))
        if bg[3] > 0:
            cr.set_source_rgba(*bg)
            cr.paint()

        bar = _parse_color(self.data.BAR_COLOR, (0.0, 1.0, 0.0, 1.0))
        grad_c = None
        if self.data.GRADIENT_COLOR:
            gc = _parse_color(self.data.GRADIENT_COLOR, (0, 0, 0, 0))
            if gc[3] > 0:
                grad_c = gc

        radius = self.data.CORNER_RADIUS or 0
        ratio = 0.75
        if self.data.REVERT_VALUE:
            ratio = 1.0 - ratio

        def _fill(fx, fy, fw, fh, color):
            if fw <= 0 or fh <= 0 or color is None:
                return
            if grad_c:
                pat = cairo.LinearGradient(fx, fy, fx, fy + fh)
                pat.add_color_stop_rgba(0, *grad_c)
                pat.add_color_stop_rgba(1, *color)
                cr.set_source(pat)
            else:
                cr.set_source_rgba(*color)
            if radius > 0:
                _rounded_rect(cr, fx, fy, fw, fh, radius)
            else:
                cr.rectangle(fx, fy, fw, fh)
            cr.fill()

        # Steps / blocks logic — mirrors Go's renderGGGraph
        steps = self.data.STEPS or 0
        block_w = self.data.BLOCK_WIDTH or 0
        step_gap = self.data.STEP_GAP or 0
        if block_w > 0:
            steps = max(1, int(w) // max(1, block_w + step_gap))

        if steps > 1 and step_gap > 0:
            seg_w = block_w if block_w > 0 else max(1, (int(w) - (steps - 1) * step_gap) // steps)
            filled_steps = round(ratio * steps)
            for i in range(steps):
                sx = i * (seg_w + step_gap)
                color = bar if i < filled_steps else (bg if bg[3] > 0 else None)
                _fill(sx, 0, seg_w, h, color)
        else:
            filled_w = round(ratio * w)
            if bg[3] > 0:
                _fill(filled_w, 0, w - filled_w, h, bg)
            _fill(0, 0, filled_w, h, bar)

        # Border
        bw = self.data.BORDER_WIDTH or 0
        if bw > 0:
            cr.set_source_rgba(*bar)
            for i in range(bw):
                cr.set_line_width(1)
                cr.rectangle(i + 0.5, i + 0.5, w - 2 * i - 1, h - 2 * i - 1)
                cr.stroke()
        elif self.data.BAR_OUTLINE:
            cr.set_source_rgba(*bar)
            cr.set_line_width(1)
            cr.rectangle(0.5, 0.5, w - 1, h - 1)
            cr.stroke()

        self._draw_selection(cr, w, h)

    def _commit_position(self):
        self.data.X = int(self._x)
        self.data.Y = int(self._y)


# ──────────────────────────────────────────────────────────────────────────────
# DraggableRadial
# ──────────────────────────────────────────────────────────────────────────────

class DraggableRadial(_DraggableBase):

    def __init__(self, data: Radial, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        size = (data.RADIUS or 80) * 2
        self.widget.set_size_request(size, size)
        r = data.RADIUS or 80
        self.set_position(data.X - r, data.Y - r)

    def invalidate(self):
        self.widget.queue_draw()

    def get_model_position(self) -> tuple[int, int]:
        r = self.data.RADIUS or 80
        return int(self._x + r), int(self._y + r)

    def _draw(self, area, cr, w, h):
        cx, cy = w / 2, h / 2
        r = self.data.RADIUS or 80
        ring_w = self.data.WIDTH or 10
        track_r = r - ring_w / 2 - 1

        # Go formula: amin = AngleStart-90, amax = 180+AngleStart-90+AngleEnd
        angle_start = math.radians(self.data.ANGLE_START - 90)
        angle_end   = math.radians(180 + self.data.ANGLE_START - 90 + (self.data.ANGLE_END or 0))
        total_arc   = angle_end - angle_start

        ratio = 0.75
        if self.data.REVERT_VALUE:
            cur = angle_end - total_arc * ratio
        else:
            cur = angle_start + total_arc * ratio

        bar_rgba = _parse_color(self.data.BAR_COLOR, (0.0, 1.0, 0.0, 1.0))
        empty_rgba = None
        if self.data.BACKGROUND_COLOR:
            c = _parse_color(self.data.BACKGROUND_COLOR, (0, 0, 0, 0))
            if c[3] > 0:
                empty_rgba = c

        if self.data.REVERT:
            bar_rgba, empty_rgba = empty_rgba, bar_rgba

        cr.set_line_cap(cairo.LINE_CAP_ROUND if self.data.ROUND else cairo.LINE_CAP_BUTT)
        cr.set_line_width(ring_w)

        def draw_arc(a1, a2, color):
            if color is None or a2 <= a1:
                return
            cr.set_source_rgba(*color)
            if self.data.CLOCKWISE:
                cr.arc(cx, cy, track_r, a1, a2)
            else:
                cr.arc_negative(cx, cy, track_r, a1, a2)
            cr.stroke()

        # Segmented arc (ANGLE_STEPS or BLOCK_ANGLE)
        angle_steps = self.data.ANGLE_STEPS or 0
        if (self.data.BLOCK_ANGLE or 0) > 0:
            block_rad = math.radians(self.data.BLOCK_ANGLE)
            angle_steps = max(1, round(total_arc / block_rad))

        if angle_steps > 1 and (self.data.ANGLE_SEP or 0) > 0:
            seg_arc = total_arc / angle_steps
            sep_arc = math.radians(self.data.ANGLE_SEP)
            filled_steps = round(ratio * angle_steps)
            for i in range(angle_steps):
                seg_start = angle_start + i * seg_arc
                seg_end   = seg_start + seg_arc - sep_arc
                if seg_end <= seg_start:
                    continue
                if self.data.REVERT_VALUE:
                    color = bar_rgba if i >= angle_steps - filled_steps else empty_rgba
                else:
                    color = bar_rgba if i < filled_steps else empty_rgba
                draw_arc(seg_start, seg_end, color)
        else:
            if self.data.REVERT_VALUE:
                draw_arc(angle_start, cur, empty_rgba)
                draw_arc(cur, angle_end, bar_rgba)
            else:
                draw_arc(angle_start, angle_end, empty_rgba)
                draw_arc(angle_start, cur, bar_rgba)

        # Center text
        if self.data.SHOW_TEXT:
            text = "75%"
            ctx = PangoCairo.font_map_get_default().create_context()
            PangoCairo.context_set_resolution(ctx, 96.0)
            layout = Pango.Layout(ctx)
            layout.set_text(text, -1)
            fd = Pango.FontDescription()
            fd.set_absolute_size(max(10, r // 3) * Pango.SCALE)
            if self.data.FONT:
                stem = os.path.splitext(os.path.basename(self.data.FONT))[0]
                fd.set_family(stem.split("-")[0])
            layout.set_font_description(fd)
            ink, logical = layout.get_pixel_extents()
            tx = cx - logical.width / 2
            ty = cy - logical.height / 2
            font_color = _parse_color(self.data.FONT_COLOR, (1, 1, 1, 1))
            cr.set_source_rgba(*font_color)
            cr.move_to(tx, ty)
            PangoCairo.show_layout(cr, layout)

        self._draw_selection(cr, w, h)

    def _commit_position(self):
        r = self.data.RADIUS or 80
        self.data.X = int(self._x + r)
        self.data.Y = int(self._y + r)


# ──────────────────────────────────────────────────────────────────────────────
# DraggableChart
# ──────────────────────────────────────────────────────────────────────────────

class DraggableChart(_DraggableBase):
    _PREVIEW_VALUES = [0.4, 0.7, 0.5, 0.8, 0.6, 0.3, 0.9, 0.55, 0.65, 0.45, 0.75, 0.35, 0.85, 0.5, 0.7]

    def __init__(self, data: Chart, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        self.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 100)
        self.set_position(data.X, data.Y)

    def invalidate(self):
        self.widget.queue_draw()

    def _draw(self, area, cr, w, h):
        cr.set_source_rgba(0.1, 0.1, 0.1, 1.0)
        cr.paint()

        col_w = self.data.COLUMN_WIDTH or 4
        gap   = self.data.COLUMN_GAP or 1
        step  = col_w + gap

        fill = _parse_color(self.data.FILL_COLOR, (0.0, 0.8, 0.0, 1.0))
        line = _parse_color(self.data.LINE_COLOR, (0.0, 1.0, 0.0, 1.0))

        style = (self.data.STYLE or "BAR").upper()
        vals  = self._PREVIEW_VALUES

        if style == "LINE":
            cr.set_source_rgba(*line)
            cr.set_line_width(1.5)
            for i, v in enumerate(vals):
                x = i * step
                y = h - v * h
                if i == 0:
                    cr.move_to(x, y)
                else:
                    cr.line_to(x, y)
            cr.stroke()
        else:
            cr.set_source_rgba(*fill)
            for i, v in enumerate(vals):
                x = i * step
                if x + col_w > w:
                    break
                bar_h = v * h
                cr.rectangle(x, h - bar_h, col_w, bar_h)
            cr.fill()

        # Border
        bw = self.data.BORDER_WIDTH or 0
        if bw > 0:
            cr.set_source_rgba(*line)
            cr.set_line_width(bw)
            cr.rectangle(bw / 2, bw / 2, w - bw, h - bw)
            cr.stroke()

        self._draw_selection(cr, w, h)

    def _commit_position(self):
        self.data.X = int(self._x)
        self.data.Y = int(self._y)


# ──────────────────────────────────────────────────────────────────────────────
# DraggableGauge
# ──────────────────────────────────────────────────────────────────────────────

class DraggableGauge(_DraggableBase):

    def __init__(self, data: Gauge, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        size = (data.RADIUS or 80) * 2
        self.widget.set_size_request(size, size)
        r = data.RADIUS or 80
        self.set_position(data.X - r, data.Y - r)

    def invalidate(self):
        self.widget.queue_draw()

    def get_model_position(self) -> tuple[int, int]:
        r = self.data.RADIUS or 80
        return int(self._x + r), int(self._y + r)

    def _draw(self, area, cr, w, h):
        cx, cy = w / 2, h / 2
        r = self.data.RADIUS or 80

        bg = _parse_color(self.data.BACKGROUND_COLOR, (0, 0, 0, 0))
        if bg[3] > 0:
            cr.set_source_rgba(*bg)
            cr.arc(cx, cy, r, 0, 2 * math.pi)
            cr.fill()

        amin = math.radians(self.data.ANGLE_START - 90)
        amax = math.radians(180 + self.data.ANGLE_START - 90 + (self.data.ANGLE_END or 0))
        cr.set_source_rgba(0.3, 0.3, 0.3, 1.0)
        cr.set_line_width(2)
        cr.arc(cx, cy, r - 4, amin, amax)
        cr.stroke()

        ratio = 0.75
        angle = amin + (amax - amin) * ratio
        needle_r = r - 6
        nx = cx + needle_r * math.cos(angle)
        ny = cy + needle_r * math.sin(angle)
        needle_color = _parse_color(self.data.NEEDLE_COLOR, (1.0, 0.0, 0.0, 1.0))
        cr.set_source_rgba(*needle_color)
        cr.set_line_width(self.data.NEEDLE_WIDTH or 3)
        cr.set_line_cap(cairo.LINE_CAP_ROUND)
        cr.move_to(cx, cy)
        cr.line_to(nx, ny)
        cr.stroke()

        cr.arc(cx, cy, max(3, (self.data.NEEDLE_WIDTH or 3)), 0, 2 * math.pi)
        cr.fill()

        self._draw_selection(cr, w, h)

    def _commit_position(self):
        r = self.data.RADIUS or 80
        self.data.X = int(self._x + r)
        self.data.Y = int(self._y + r)


# ──────────────────────────────────────────────────────────────────────────────
# DraggableStatusBar
# ──────────────────────────────────────────────────────────────────────────────

class DraggableStatusBar(_DraggableBase):

    def __init__(self, data: StatusBar, yaml_path: str,
                 canvas_w: int = 1280, canvas_h: int = 720):
        super().__init__(canvas_w, canvas_h)
        self.data = data
        self.yaml_path = yaml_path
        self.widget.set_size_request(data.WIDTH or 200, data.HEIGHT or 20)
        self.set_position(data.X, data.Y)

    def invalidate(self):
        self.widget.queue_draw()

    def _draw(self, area, cr, w, h):
        bg = _parse_color(self.data.BACKGROUND_COLOR, (0.15, 0.15, 0.15, 1.0))
        if bg[3] > 0:
            cr.set_source_rgba(*bg)
            cr.rectangle(0, 0, w, h)
            cr.fill()

        ratio = 0.75
        bar_w = ratio * w
        bar = _parse_color(self.data.BAR_COLOR, (0.0, 1.0, 0.0, 1.0))
        cr.set_source_rgba(*bar)
        cr.rectangle(0, 0, bar_w, h)
        cr.fill()

        ind_r = self.data.INDICATOR_RADIUS or max(3, int(h / 2))
        ind_color = _parse_color(self.data.INDICATOR_COLOR, (1.0, 1.0, 1.0, 1.0))
        cr.set_source_rgba(*ind_color)
        cr.arc(bar_w, h / 2, ind_r, 0, 2 * math.pi)
        cr.fill()

        self._draw_selection(cr, w, h)

    def _commit_position(self):
        self.data.X = int(self._x)
        self.data.Y = int(self._y)
