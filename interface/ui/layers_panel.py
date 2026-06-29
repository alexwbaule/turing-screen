import os
from gi.repository import Gtk, GLib


class LayersPanel(Gtk.Box):
    """
    Lists background layers + all canvas elements.
    Click → selects that element on the canvas.
    Up/Down → reorders layers.
    Add/Remove → add/remove background images.
    """

    def __init__(self, canvas):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=4)
        self._canvas = canvas
        self._entries = []      # list of dicts: {label, type, key/widget}
        self._selected_index = -1

        self._build_ui()

    def _build_ui(self):
        # Header
        header = Gtk.Label(label="Layers")
        header.add_css_class("heading")
        header.set_halign(Gtk.Align.START)
        self.append(header)

        # Toolbar
        tb = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=2)
        for icon, tip, cb in [
            ("list-add-symbolic",      "Add layer",    self._on_add),
            ("list-remove-symbolic",   "Remove",       self._on_remove),
            ("go-up-symbolic",         "Move up",      self._on_move_up),
            ("go-down-symbolic",       "Move down",    self._on_move_down),
        ]:
            btn = Gtk.Button()
            btn.set_child(Gtk.Image.new_from_icon_name(icon))
            btn.set_tooltip_text(tip)
            btn.connect("clicked", cb)
            tb.append(btn)
        self.append(tb)

        # ListBox
        scroll = Gtk.ScrolledWindow()
        scroll.set_vexpand(True)
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)

        self._listbox = Gtk.ListBox()
        self._listbox.set_selection_mode(Gtk.SelectionMode.SINGLE)
        self._listbox.connect("row-selected", self._on_row_selected)
        scroll.set_child(self._listbox)
        self.append(scroll)

    # ------------------------------------------------------------------
    # Public API called by canvas
    # ------------------------------------------------------------------

    def refresh(self, theme, canvas_elements: list):
        """Rebuild list from current theme + canvas elements."""
        self._entries.clear()
        while self._listbox.get_first_child():
            self._listbox.remove(self._listbox.get_first_child())

        # Video (if present)
        if theme and theme.video:
            self._add_entry("🎬 VIDEO", "video", None)

        # Background layers (in sorted key order)
        if theme and theme.static_images:
            for key in sorted(theme.static_images.keys()):
                img = theme.static_images[key]
                label = f"🖼 {key}  ({os.path.basename(img.PATH)})"
                self._add_entry(label, "layer", key)

        # Canvas elements
        for elem in canvas_elements:
            label = f"📝 {elem.yaml_path}"
            self._add_entry(label, "widget", elem)

    def select_element(self, elem):
        """Highlight the row matching elem (called from canvas on click)."""
        for i, entry in enumerate(self._entries):
            if entry.get("widget") is elem:
                self._set_selected(i, notify_canvas=False)
                return

    def select_bg(self, key):
        """Highlight the background-layer row matching key (no canvas callback)."""
        for i, entry in enumerate(self._entries):
            if entry.get("kind") == "layer" and entry.get("key") == key:
                self._set_selected(i, notify_canvas=False)
                return

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _add_entry(self, label: str, kind: str, ref):
        entry = {"label": label, "kind": kind}
        if kind == "layer":
            entry["key"] = ref
        elif kind == "widget":
            entry["widget"] = ref
        self._entries.append(entry)

        lbl = Gtk.Label(label=label)
        lbl.set_halign(Gtk.Align.START)
        lbl.set_margin_top(3)
        lbl.set_margin_bottom(3)
        lbl.set_margin_start(6)
        row = Gtk.ListBoxRow()
        row.set_child(lbl)
        self._listbox.append(row)

    def _set_selected(self, index: int, notify_canvas: bool = True):
        self._selected_index = index
        rows = self._get_all_rows()
        if 0 <= index < len(rows):
            self._listbox.select_row(rows[index])
        if notify_canvas and 0 <= index < len(self._entries):
            entry = self._entries[index]
            if entry["kind"] == "widget":
                self._canvas.select_element(entry["widget"])
            elif entry["kind"] == "layer":
                self._canvas.select_bg_image(entry["key"])
            else:
                self._canvas.select_element(None)

    def _get_all_rows(self) -> list:
        rows = []
        r = self._listbox.get_first_child()
        while r:
            rows.append(r)
            r = r.get_next_sibling()
        return rows

    def _on_row_selected(self, listbox, row):
        if row is None:
            return
        rows = self._get_all_rows()
        try:
            idx = rows.index(row)
        except ValueError:
            return
        self._set_selected(idx)

    def _on_add(self, _btn):
        dialog = Gtk.FileDialog()
        dialog.set_title("Select background image (PNG)")
        filt = Gtk.FileFilter()
        filt.set_name("PNG images")
        filt.add_pattern("*.png")
        # Simple approach: use open() without filter for now
        dialog.open(self._canvas.get_root(), None, self._on_file_chosen)

    def _on_file_chosen(self, dialog, result):
        try:
            from gi.repository import Gio
            gfile = dialog.open_finish(result)
            path = gfile.get_path()
            self._canvas.add_background_layer(path)
        except Exception:
            pass

    def _on_remove(self, _btn):
        if self._selected_index < 0 or self._selected_index >= len(self._entries):
            return
        entry = self._entries[self._selected_index]
        if entry["kind"] == "layer":
            self._canvas.remove_background_layer(entry["key"])
        elif entry["kind"] == "widget":
            self._canvas.remove_element(entry["widget"])

    def _on_move_up(self, _btn):
        i = self._selected_index
        if i <= 0:
            return
        self._canvas.reorder_layer(i, i - 1)

    def _on_move_down(self, _btn):
        i = self._selected_index
        if i < 0 or i >= len(self._entries) - 1:
            return
        self._canvas.reorder_layer(i, i + 1)
