"""
Toolbox — painel esquerdo do editor de temas.

Dividido em dois painéis separados por um separador (espelhando Properties+Layers
no lado direito):

  ┌─────────────────┐
  │  Add Element    │  ← accordion por categoria; cada sensor é um botão
  ├─────────────────┤
  │  Representação  │  ← radio buttons para escolher o tipo de exibição
  └─────────────────┘

Clicar num sensor cria um elemento do tipo atualmente selecionado na
seção "Representação". Sensores text-only (Load, Condition, Volume…)
ignoram o seletor e sempre criam Text.
"""
from gi.repository import Gtk

# ---------------------------------------------------------------------------
# Mapeamento tipo → (sufixo YAML, kind para canvas)
# ---------------------------------------------------------------------------
_TYPES = [
    ("Text",        "TEXT",        "text"),
    ("% Text",      "PERCENT_TEXT","text"),
    ("Graph",       "GRAPH",       "graph"),
    ("Radial",      "RADIAL",      "radial"),
    ("Chart",       "CHART",       "chart"),
    ("Gauge",       "GAUGE",       "gauge"),
    ("Status Bar",  "STATUS_BAR",  "status_bar"),
]

# ---------------------------------------------------------------------------
# Definição das seções do accordion
# Cada sensor: (label, base_path, text_only)
#   text_only=True  → base_path já é o caminho completo; ignora tipo selecionado
#   text_only=False → base_path + "." + sufixo do tipo selecionado
# ---------------------------------------------------------------------------
# Network/Frequency use a float formatter (NetSpeed, Hertz, Bytes) — Gauge and StatusBar
# can still be used (scale via MIN/MAX_VALUE), but PercentText has no clear meaning here.
_FLOAT_SUFFIXES = frozenset({"TEXT", "GRAPH", "RADIAL", "CHART", "GAUGE", "STATUS_BAR"})

# Tuple layout: (label, base_path, text_only, allowed_suffixes)
#   text_only=True        → always creates Text; allowed_suffixes ignored
#   text_only=False       → appends type suffix; None = all types allowed
#   allowed_suffixes      → frozenset of YAML suffixes valid for this sensor,
#                           or None to allow every type in _TYPES
_SECTIONS = [
    ("Texto Estático", [
        ("Texto",        "static_texts.LABEL",      True, None),
        ("CPU Model",    "static_texts.CPU_MODEL",   True, None),
        ("GPU Model",    "static_texts.GPU_MODEL",   True, None),
        ("Total RAM",    "static_texts.MEM_TOTAL",   True, None),
        ("Modelo Disco", "static_texts.DISK_MODEL",  True, None),
        ("Hostname",     "static_texts.HOSTNAME",    True, None),
    ]),
    ("CPU", [
        ("Modelo",      "STATS.CPU.MODEL.TEXT",            True,  None),
        ("Percentage",  "STATS.CPU.PERCENTAGE",            False, None),
        ("Temperature", "STATS.CPU.TEMPERATURE",           False, None),
        # Frequency uses collectMeasurementFloatItems — no Gauge/StatusBar/PercentText
        ("Frequency",   "STATS.CPU.FREQUENCY",             False, _FLOAT_SUFFIXES),
        ("Fan",         "STATS.CPU.FAN",                   False, None),
        ("Power",       "STATS.CPU.POWER",                 False, None),
        ("Voltage",     "STATS.CPU.VOLTAGE",               False, None),
        ("Load 1min",   "STATS.CPU.LOAD.ONE",              False, None),
        ("Load 5min",   "STATS.CPU.LOAD.FIVE",             False, None),
        ("Load 15min",  "STATS.CPU.LOAD.FIFTEEN",          False, None),
    ]),
    ("GPU", [
        ("Modelo",      "STATS.GPU.MODEL.TEXT",            True,  None),
        ("Percentage",  "STATS.GPU.PERCENTAGE",            False, None),
        ("Memory",      "STATS.GPU.MEMORY",                False, None),
        ("Temperature", "STATS.GPU.TEMPERATURE",           False, None),
        ("Power",       "STATS.GPU.POWER",                 False, None),
        # Frequency uses collectMeasurementFloatItems — no Gauge/StatusBar/PercentText
        ("Frequency",   "STATS.GPU.FREQUENCY",             False, _FLOAT_SUFFIXES),
        ("Voltage",     "STATS.GPU.VOLTAGE",               False, None),
        ("Fan",         "STATS.GPU.FAN",                   False, None),
    ]),
    ("RAM", [
        ("Uso",     "STATS.MEMORY.RAM",                   False, None),
        ("% Texto", "STATS.MEMORY.RAM.PERCENT_TEXT",      True,  None),
        ("Modelo",  "STATS.MEMORY.RAM.MODEL.TEXT",        True,  None),
        ("Tamanho", "STATS.MEMORY.RAM.SIZE.TEXT",         True,  None),
    ]),
    ("Swap", [
        ("Uso",     "STATS.MEMORY.SWAP",              False, None),
        ("% Texto", "STATS.MEMORY.SWAP.PERCENT_TEXT", True,  None),
    ]),
    ("Disco", [
        ("Usado",       "STATS.DISK.USED",         False, None),
        ("Livre",       "STATS.DISK.FREE",         False, None),
        # TOTAL exists in the entity but is NOT rendered by the compositor
        ("Temperatura", "STATS.DISK.TEMPERATURE",  False, None),
    ]),
    ("Rede Ethernet", [
        ("Upload",     "STATS.NET.ETH.UPLOAD",     False, _FLOAT_SUFFIXES),
        ("Download",   "STATS.NET.ETH.DOWNLOAD",   False, _FLOAT_SUFFIXES),
        ("Total Up",   "STATS.NET.ETH.UPLOADED",   False, _FLOAT_SUFFIXES),
        ("Total Down", "STATS.NET.ETH.DOWNLOADED", False, _FLOAT_SUFFIXES),
    ]),
    ("Rede WiFi", [
        ("Upload",     "STATS.NET.WLO.UPLOAD",     False, _FLOAT_SUFFIXES),
        ("Download",   "STATS.NET.WLO.DOWNLOAD",   False, _FLOAT_SUFFIXES),
        ("Total Up",   "STATS.NET.WLO.UPLOADED",   False, _FLOAT_SUFFIXES),
        ("Total Down", "STATS.NET.WLO.DOWNLOADED", False, _FLOAT_SUFFIXES),
    ]),
    ("Data / Hora", [
        ("Hora", "STATS.DATE.HOUR", False, None),
        ("Dia",  "STATS.DATE.DAY",  False, None),
    ]),
    ("Clima", [
        ("Temperatura", "STATS.WEATHER.TEMPERATURE", False, None),
        ("Condição",    "STATS.WEATHER.CONDITION",   True,  None),
    ]),
    ("Volume", [
        ("Volume", "STATS.VOLUME", False, None),
    ]),
]


class Toolbox(Gtk.Box):
    """
    Left sidebar: sensor accordion (top) + representation type picker (bottom).
    """

    def __init__(self, add_element_cb):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        self.set_size_request(180, -1)
        self._add_cb = add_element_cb
        self._selected_type_idx = 0   # index into _TYPES (default: Text)
        self._type_buttons: list[tuple] = []  # (CheckButton, suffix, kind)
        self._build_ui()

    # ------------------------------------------------------------------

    def _build_ui(self):
        self._build_sensor_panel()
        self.append(Gtk.Separator())
        self._build_type_panel()

    # ── Sensor accordion ──────────────────────────────────────────────

    def _build_sensor_panel(self):
        header = Gtk.Label(label="Add Element")
        header.add_css_class("heading")
        header.set_halign(Gtk.Align.START)
        header.set_margin_top(8)
        header.set_margin_start(8)
        header.set_margin_bottom(4)
        self.append(header)

        scroll = Gtk.ScrolledWindow()
        scroll.set_vexpand(True)
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)

        inner = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)

        for section_title, sensors in _SECTIONS:
            exp = Gtk.Expander(label=section_title)
            exp.set_margin_start(4)
            exp.set_margin_top(2)

            vbox = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
            vbox.set_margin_start(12)
            vbox.set_margin_top(4)
            vbox.set_margin_bottom(4)

            for label, base_path, text_only, allowed_suffixes in sensors:
                btn = Gtk.Button(label=label)
                btn.set_halign(Gtk.Align.START)
                btn.add_css_class("flat")
                btn.connect("clicked", self._make_sensor_handler(
                    base_path, text_only, allowed_suffixes))
                vbox.append(btn)

            exp.set_child(vbox)
            inner.append(exp)

        scroll.set_child(inner)
        self.append(scroll)

    def _update_type_sensitivity(self, allowed_suffixes):
        """Enable/disable type radio buttons based on what the sensor supports.

        If the currently selected type is not allowed, switches to the first
        allowed type so the next click creates a valid element."""
        if not self._type_buttons:
            return
        if allowed_suffixes is None:
            for rb, _s, _k in self._type_buttons:
                rb.set_sensitive(True)
            return
        # Determine if the current selection is still valid
        _, cur_suffix, _ = _TYPES[self._selected_type_idx]
        first_valid_idx = None
        for i, (rb, suffix, _kind) in enumerate(self._type_buttons):
            valid = suffix in allowed_suffixes
            rb.set_sensitive(valid)
            if valid and first_valid_idx is None:
                first_valid_idx = i
        if cur_suffix not in allowed_suffixes and first_valid_idx is not None:
            # Auto-select the first valid type
            self._type_buttons[first_valid_idx][0].set_active(True)
            self._selected_type_idx = first_valid_idx

    def _make_sensor_handler(self, base_path: str, text_only: bool,
                             allowed_suffixes):
        def handler(_btn):
            # Update representation picker sensitivity for this sensor category
            self._update_type_sensitivity(allowed_suffixes)
            if text_only:
                self._add_cb(base_path, "text")
            else:
                _, suffix, kind = _TYPES[self._selected_type_idx]
                self._add_cb(base_path + "." + suffix, kind)
        return handler

    # ── Type picker ───────────────────────────────────────────────────

    def _build_type_panel(self):
        header = Gtk.Label(label="Representação")
        header.add_css_class("heading")
        header.set_halign(Gtk.Align.START)
        header.set_margin_top(8)
        header.set_margin_start(8)
        header.set_margin_bottom(4)
        self.append(header)

        vbox = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
        vbox.set_margin_start(8)
        vbox.set_margin_end(8)
        vbox.set_margin_bottom(8)

        first_btn = None
        self._type_buttons = []
        for i, (label, suffix, kind) in enumerate(_TYPES):
            rb = Gtk.CheckButton(label=label)
            if first_btn is None:
                first_btn = rb
                rb.set_active(True)
            else:
                rb.set_group(first_btn)
            rb.connect("toggled", self._make_type_handler(i))
            vbox.append(rb)
            self._type_buttons.append((rb, suffix, kind))

        self.append(vbox)

    def _make_type_handler(self, idx: int):
        def handler(btn):
            if btn.get_active():
                self._selected_type_idx = idx
        return handler
