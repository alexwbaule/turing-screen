"""
Toolbox — buttons to add new elements to the canvas.
One section per sensor category (CPU, GPU, Memory, Disk, Net, Date, Static).
"""
from gi.repository import Gtk


# Maps (label, yaml_path_hint, element_type) per category
_SECTIONS = [
    ("Texto Estático", [
        ("Texto Estático", "static_texts.LABEL", "text"),
    ]),
    ("CPU", [
        ("% Graph",       "STATS.CPU.PERCENTAGE.GRAPH",      "graph"),
        ("% Text",        "STATS.CPU.PERCENTAGE.TEXT",       "text"),
        ("% Radial",      "STATS.CPU.PERCENTAGE.RADIAL",     "radial"),
        ("% Chart",       "STATS.CPU.PERCENTAGE.CHART",      "chart"),
        ("Temperatura",   "STATS.CPU.TEMPERATURE.TEXT",      "text"),
        ("Frequência",    "STATS.CPU.FREQUENCY.TEXT",        "text"),
        ("Fan",           "STATS.CPU.FAN.TEXT",              "text"),
        ("Power",         "STATS.CPU.POWER.TEXT",            "text"),
        ("Voltage",       "STATS.CPU.VOLTAGE.TEXT",          "text"),
        ("Load 1min",     "STATS.CPU.LOAD.ONE.TEXT",         "text"),
        ("Load 5min",     "STATS.CPU.LOAD.FIVE.TEXT",        "text"),
        ("Load 15min",    "STATS.CPU.LOAD.FIFTEEN.TEXT",     "text"),
    ]),
    ("GPU", [
        ("% Graph",       "STATS.GPU.PERCENTAGE.GRAPH",      "graph"),
        ("% Text",        "STATS.GPU.PERCENTAGE.TEXT",       "text"),
        ("% Radial",      "STATS.GPU.PERCENTAGE.RADIAL",     "radial"),
        ("% Chart",       "STATS.GPU.PERCENTAGE.CHART",      "chart"),
        ("Temperatura",   "STATS.GPU.TEMPERATURE.TEXT",      "text"),
        ("Memória",       "STATS.GPU.MEMORY.TEXT",           "text"),
        ("Power",         "STATS.GPU.POWER.TEXT",            "text"),
        ("Frequência",    "STATS.GPU.FREQUENCY.TEXT",        "text"),
        ("Voltage",       "STATS.GPU.VOLTAGE.TEXT",          "text"),
        ("Fan",           "STATS.GPU.FAN.TEXT",              "text"),
    ]),
    ("Memória Virtual", [
        ("% Text",        "STATS.MEMORY.VIRTUAL.PERCENT_TEXT", "text"),
        ("Graph",         "STATS.MEMORY.VIRTUAL.GRAPH",        "graph"),
        ("Radial",        "STATS.MEMORY.VIRTUAL.RADIAL",       "radial"),
        ("Chart",         "STATS.MEMORY.VIRTUAL.CHART",        "chart"),
        ("Usada",         "STATS.MEMORY.VIRTUAL.USED",         "text"),
        ("Livre",         "STATS.MEMORY.VIRTUAL.FREE",         "text"),
    ]),
    ("Swap", [
        ("% Text",        "STATS.MEMORY.SWAP.PERCENT_TEXT",    "text"),
        ("Graph",         "STATS.MEMORY.SWAP.GRAPH",           "graph"),
        ("Radial",        "STATS.MEMORY.SWAP.RADIAL",          "radial"),
        ("Chart",         "STATS.MEMORY.SWAP.CHART",           "chart"),
    ]),
    ("Disco", [
        ("Usado %",       "STATS.DISK.USED.PERCENT_TEXT",      "text"),
        ("Livre",         "STATS.DISK.FREE.TEXT",              "text"),
        ("Total",         "STATS.DISK.TOTAL.TEXT",             "text"),
        ("Temperatura",   "STATS.DISK.TEMPERATURE.TEXT",       "text"),
    ]),
    ("Rede Ethernet", [
        ("Upload",        "STATS.NET.ETH.UPLOAD.TEXT",         "text"),
        ("Download",      "STATS.NET.ETH.DOWNLOAD.TEXT",       "text"),
        ("Total Up",      "STATS.NET.ETH.UPLOADED.TEXT",       "text"),
        ("Total Down",    "STATS.NET.ETH.DOWNLOADED.TEXT",     "text"),
    ]),
    ("Rede WiFi", [
        ("Upload",        "STATS.NET.WLO.UPLOAD.TEXT",         "text"),
        ("Download",      "STATS.NET.WLO.DOWNLOAD.TEXT",       "text"),
        ("Total Up",      "STATS.NET.WLO.UPLOADED.TEXT",       "text"),
        ("Total Down",    "STATS.NET.WLO.DOWNLOADED.TEXT",     "text"),
    ]),
    ("Data / Hora", [
        ("Hora",          "STATS.DATE.HOUR.TEXT",              "text"),
        ("Dia",           "STATS.DATE.DAY.TEXT",               "text"),
    ]),
    ("Clima", [
        ("Temperatura",   "STATS.WEATHER.TEMPERATURE.TEXT",    "text"),
        ("Condição",      "STATS.WEATHER.CONDITION",           "text"),
    ]),
    ("Volume", [
        ("Volume",        "STATS.VOLUME.TEXT",                 "text"),
    ]),
]


class Toolbox(Gtk.Box):
    """
    Left sidebar with collapsible sections. Each button calls
    add_element_cb(yaml_path, element_type).
    """

    def __init__(self, add_element_cb):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        self.set_size_request(180, -1)
        self._add_cb = add_element_cb
        self._build_ui()

    def _build_ui(self):
        lbl = Gtk.Label(label="Add Element")
        lbl.add_css_class("heading")
        lbl.set_halign(Gtk.Align.START)
        lbl.set_margin_top(8)
        lbl.set_margin_start(8)
        lbl.set_margin_bottom(4)
        self.append(lbl)

        scroll = Gtk.ScrolledWindow()
        scroll.set_vexpand(True)
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)

        inner = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)

        for section_title, items in _SECTIONS:
            expander = Gtk.Expander(label=section_title)
            expander.set_margin_start(4)
            expander.set_margin_top(2)

            vbox = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
            vbox.set_margin_start(12)
            vbox.set_margin_top(4)
            vbox.set_margin_bottom(4)

            for label, path, kind in items:
                btn = Gtk.Button(label=label)
                btn.set_halign(Gtk.Align.START)
                btn.add_css_class("flat")
                # capture loop vars
                btn.connect("clicked", self._make_handler(path, kind))
                vbox.append(btn)

            expander.set_child(vbox)
            inner.append(expander)

        scroll.set_child(inner)
        self.append(scroll)

    def _make_handler(self, path: str, kind: str):
        def handler(_btn):
            self._add_cb(path, kind)
        return handler
