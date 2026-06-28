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
_SECTIONS = [
    ("Texto Estático", [
        ("Texto",       "static_texts.LABEL",              True),
    ]),
    ("CPU", [
        ("Percentage",  "STATS.CPU.PERCENTAGE",            False),
        ("Temperature", "STATS.CPU.TEMPERATURE",           False),
        ("Frequency",   "STATS.CPU.FREQUENCY",             False),
        ("Fan",         "STATS.CPU.FAN",                   False),
        ("Power",       "STATS.CPU.POWER",                 False),
        ("Voltage",     "STATS.CPU.VOLTAGE",               False),
        ("Load 1min",   "STATS.CPU.LOAD.ONE.TEXT",         True),
        ("Load 5min",   "STATS.CPU.LOAD.FIVE.TEXT",        True),
        ("Load 15min",  "STATS.CPU.LOAD.FIFTEEN.TEXT",     True),
    ]),
    ("GPU", [
        ("Percentage",  "STATS.GPU.PERCENTAGE",            False),
        ("Memory",      "STATS.GPU.MEMORY",                False),
        ("Temperature", "STATS.GPU.TEMPERATURE",           False),
        ("Power",       "STATS.GPU.POWER",                 False),
        ("Frequency",   "STATS.GPU.FREQUENCY",             False),
        ("Voltage",     "STATS.GPU.VOLTAGE",               False),
        ("Fan",         "STATS.GPU.FAN",                   False),
    ]),
    ("Memória Virtual", [
        ("Uso",         "STATS.MEMORY.VIRTUAL",            False),
        ("Usada",       "STATS.MEMORY.VIRTUAL.USED",       True),
        ("Livre",       "STATS.MEMORY.VIRTUAL.FREE",       True),
        ("% Texto",     "STATS.MEMORY.VIRTUAL.PERCENT_TEXT", True),
    ]),
    ("Swap", [
        ("Uso",         "STATS.MEMORY.SWAP",               False),
        ("% Texto",     "STATS.MEMORY.SWAP.PERCENT_TEXT",  True),
    ]),
    ("Disco", [
        ("Usado",       "STATS.DISK.USED",                 False),
        ("Livre",       "STATS.DISK.FREE",                 False),
        ("Total",       "STATS.DISK.TOTAL",                False),
        ("Temperatura", "STATS.DISK.TEMPERATURE",          False),
    ]),
    ("Rede Ethernet", [
        ("Upload",      "STATS.NET.ETH.UPLOAD",            False),
        ("Download",    "STATS.NET.ETH.DOWNLOAD",          False),
        ("Total Up",    "STATS.NET.ETH.UPLOADED",          False),
        ("Total Down",  "STATS.NET.ETH.DOWNLOADED",        False),
    ]),
    ("Rede WiFi", [
        ("Upload",      "STATS.NET.WLO.UPLOAD",            False),
        ("Download",    "STATS.NET.WLO.DOWNLOAD",          False),
        ("Total Up",    "STATS.NET.WLO.UPLOADED",          False),
        ("Total Down",  "STATS.NET.WLO.DOWNLOADED",        False),
    ]),
    ("Data / Hora", [
        ("Hora",        "STATS.DATE.HOUR",                 False),
        ("Dia",         "STATS.DATE.DAY",                  False),
    ]),
    ("Clima", [
        ("Temperatura", "STATS.WEATHER.TEMPERATURE",       False),
        ("Condição",    "STATS.WEATHER.CONDITION",         True),
    ]),
    ("Volume", [
        ("Volume",      "STATS.VOLUME.TEXT",               True),
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

            for label, base_path, text_only in sensors:
                btn = Gtk.Button(label=label)
                btn.set_halign(Gtk.Align.START)
                btn.add_css_class("flat")
                btn.connect("clicked", self._make_sensor_handler(base_path, text_only))
                vbox.append(btn)

            exp.set_child(vbox)
            inner.append(exp)

        scroll.set_child(inner)
        self.append(scroll)

    def _make_sensor_handler(self, base_path: str, text_only: bool):
        def handler(_btn):
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
        for i, (label, _suffix, _kind) in enumerate(_TYPES):
            rb = Gtk.CheckButton(label=label)
            if first_btn is None:
                first_btn = rb
                rb.set_active(True)
            else:
                rb.set_group(first_btn)
            rb.connect("toggled", self._make_type_handler(i))
            vbox.append(rb)

        self.append(vbox)

    def _make_type_handler(self, idx: int):
        def handler(btn):
            if btn.get_active():
                self._selected_type_idx = idx
        return handler
