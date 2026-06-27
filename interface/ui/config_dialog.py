"""
Tela de Configurações — espelho de config_dialog.go (Go/Fyne).
Lê e grava conf/config.yaml (mesma estrutura usada pelo daemon via viper).
Quatro abas: Geral, Display, Sensores, Clima.
"""
import os
import logging

import yaml
from gi.repository import Gtk, Adw

log = logging.getLogger(__name__)


def _config_path() -> str:
    """Resolve conf/config.yaml: cwd primeiro (daemon roda em /opt/smart-screen),
    depois relativo a este arquivo (../conf/config.yaml)."""
    cwd_cand = os.path.join(os.getcwd(), "conf", "config.yaml")
    if os.path.exists(cwd_cand):
        return cwd_cand
    base = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    return os.path.abspath(os.path.join(base, "..", "conf", "config.yaml"))


class ConfigDialog(Gtk.Window):
    """Janela modal de configurações do daemon."""

    LOG_LEVELS = ["debug", "info", "warn", "error"]
    GPU_PROVIDERS = ["auto", "nvidia", "amd", "noop"]

    def __init__(self, parent):
        super().__init__(transient_for=parent, modal=True, destroy_with_parent=True)
        self.set_title("Configurações")
        self.set_default_size(540, 560)

        self._path = _config_path()
        self._cfg = self._load()
        dev = self._cfg.setdefault("device", {})

        # ── Header bar ──
        header = Gtk.HeaderBar()
        save_btn = Gtk.Button(label="Salvar")
        save_btn.add_css_class("suggested-action")
        save_btn.connect("clicked", self._on_save)
        header.pack_end(save_btn)
        self.set_titlebar(header)

        # ── Notebook com 4 abas (equivalente ao NewAppTabs do Go) ──
        notebook = Gtk.Notebook()
        notebook.set_tab_pos(Gtk.PositionType.TOP)

        notebook.append_page(self._page_geral(dev),    Gtk.Label(label="Geral"))
        notebook.append_page(self._page_display(dev),  Gtk.Label(label="Display"))
        notebook.append_page(self._page_sensores(dev), Gtk.Label(label="Sensores"))
        notebook.append_page(self._page_clima(dev),    Gtk.Label(label="Clima"))

        self.set_child(notebook)

    # ------------------------------------------------------------------
    # YAML
    # ------------------------------------------------------------------

    def _load(self) -> dict:
        try:
            with open(self._path) as f:
                return yaml.safe_load(f) or {}
        except Exception as e:
            log.warning("Config: falha ao ler %s: %s", self._path, e)
            return {}

    def _save_yaml(self) -> bool:
        try:
            with open(self._path, "w") as f:
                yaml.safe_dump(self._cfg, f, default_flow_style=False, sort_keys=False)
            return True
        except Exception as e:
            log.error("Config: falha ao gravar %s: %s", self._path, e)
            return False

    # ------------------------------------------------------------------
    # Helpers de row
    # ------------------------------------------------------------------

    @staticmethod
    def _scrolled(widget) -> Gtk.ScrolledWindow:
        sw = Gtk.ScrolledWindow()
        sw.set_hexpand(True)
        sw.set_vexpand(True)
        sw.set_child(widget)
        sw.set_margin_start(12)
        sw.set_margin_end(12)
        sw.set_margin_top(12)
        sw.set_margin_bottom(12)
        return sw

    @staticmethod
    def _entry(dev, key_path, title, placeholder="") -> Adw.EntryRow:
        row = Adw.EntryRow()
        row.set_title(title)
        if placeholder:
            row.set_show_apply_button(False)
        val = _get_path(dev, key_path)
        if val not in (None, ""):
            row.set_text(str(val))
        if placeholder:
            row.set_text(placeholder if val in (None, "") else str(val))
        return row

    @staticmethod
    def _spin(dev, key_path, title, lo, hi, step=1) -> Adw.SpinRow:
        row = Adw.SpinRow.new_with_range(lo, hi, step)
        row.set_title(title)
        val = _get_path(dev, key_path)
        row.set_value(float(val) if val not in (None, "") else lo)
        row.set_numeric(True)
        return row

    @staticmethod
    def _combo(dev, key_path, title, options) -> Adw.ComboRow:
        row = Adw.ComboRow()
        row.set_title(title)
        model = Gtk.StringList.new(options)
        row.set_model(model)
        cur = _get_path(dev, key_path) or options[0]
        idx = options.index(cur) if cur in options else 0
        row.set_selected(idx)
        return row

    @staticmethod
    def _switch(dev, key_path, title) -> Adw.SwitchRow:
        row = Adw.SwitchRow()
        row.set_title(title)
        row.set_active(bool(_get_path(dev, key_path)))
        return row

    @staticmethod
    def _group(title) -> Adw.PreferencesGroup:
        g = Adw.PreferencesGroup()
        g.set_title(title)
        return g

    # ------------------------------------------------------------------
    # Abas
    # ------------------------------------------------------------------

    def _page_geral(self, dev) -> Gtk.ScrolledWindow:
        g = self._group("Geral")
        self._r_port = self._entry(dev, "port", "Porta Serial")
        self._r_port.set_text(str(dev.get("port", "AUTO")))
        self._r_api = self._spin(dev, "api_port", "API Port", 1, 65535)
        self._r_api.set_value(int(dev.get("api_port", 9120)))
        self._r_log = self._combo(dev, "log", "Log Level", self.LOG_LEVELS)
        self._r_tur = self._switch(dev, "turn_off_on_exit", "Desligar display ao sair")
        for r in (self._r_port, self._r_api, self._r_log, self._r_tur):
            g.add(r)
        return self._scrolled(g)

    def _page_display(self, dev) -> Gtk.ScrolledWindow:
        g = self._group("Display")
        disp = dev.setdefault("display", {})
        self._r_bright = self._spin(disp, "brightness", "Brilho", 0, 100)
        self._r_bright.set_value(int(disp.get("brightness", 25)))
        self._r_w = self._spin(disp, "width", "Largura (px)", 1, 4096)
        self._r_w.set_value(int(disp.get("width", 1280)))
        self._r_h = self._spin(disp, "height", "Altura (px)", 1, 4096)
        self._r_h.set_value(int(disp.get("height", 720)))
        self._r_rev = self._switch(disp, "reverse", "Inverter orientação")
        for r in (self._r_bright, self._r_w, self._r_h, self._r_rev):
            g.add(r)
        return self._scrolled(g)

    def _page_sensores(self, dev) -> Gtk.ScrolledWindow:
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=18)

        sensors = dev.setdefault("sensors", {})

        # CPU
        cpu = sensors.setdefault("cpu", {})
        gcpu = self._group("CPU")
        self._r_cpu_int = self._entry(cpu, "interval", "Intervalo")
        self._r_cpu_int.set_text(str(cpu.get("interval", "1s")))
        self._r_cpu_temp = self._entry(cpu, "temperature_sensor", "Sensor de Temperatura")
        self._r_cpu_temp.set_text(str(cpu.get("temperature_sensor", "auto")))
        gcpu.add(self._r_cpu_int)
        gcpu.add(self._r_cpu_temp)
        box.append(gcpu)

        # GPU
        gpu = sensors.setdefault("gpu", {})
        ggpu = self._group("GPU")
        self._r_gpu_int = self._entry(gpu, "interval", "Intervalo")
        self._r_gpu_int.set_text(str(gpu.get("interval", "1s")))
        self._r_gpu_prov = self._combo(gpu, "provider", "Provider", self.GPU_PROVIDERS)
        ggpu.add(self._r_gpu_int)
        ggpu.add(self._r_gpu_prov)
        box.append(ggpu)

        # Memória
        mem = sensors.setdefault("memory", {})
        gmem = self._group("Memória")
        self._r_mem_int = self._entry(mem, "interval", "Intervalo")
        self._r_mem_int.set_text(str(mem.get("interval", "5s")))
        gmem.add(self._r_mem_int)
        box.append(gmem)

        # Disco
        disk = sensors.setdefault("disk", {})
        gdisk = self._group("Disco")
        self._r_disk_int = self._entry(disk, "interval", "Intervalo")
        self._r_disk_int.set_text(str(disk.get("interval", "10s")))
        self._r_disk_temp = self._entry(disk, "temperature_sensor", "Sensor de Temperatura")
        self._r_disk_temp.set_text(str(disk.get("temperature_sensor", "auto")))
        gdisk.add(self._r_disk_int)
        gdisk.add(self._r_disk_temp)
        box.append(gdisk)

        # Rede
        net = sensors.setdefault("network", {})
        gnet = self._group("Rede")
        self._r_eth = self._entry(net, "eth", "Interface Cabeada")
        self._r_eth.set_text(str(net.get("eth", "enp3s0")))
        self._r_wlo = self._entry(net, "wlo", "Interface Wi-Fi")
        self._r_wlo.set_text(str(net.get("wlo", "wlp4s0")))
        self._r_net_int = self._entry(net, "interval", "Intervalo")
        self._r_net_int.set_text(str(net.get("interval", "1s")))
        gnet.add(self._r_eth)
        gnet.add(self._r_wlo)
        gnet.add(self._r_net_int)
        box.append(gnet)

        return self._scrolled(box)

    def _page_clima(self, dev) -> Gtk.ScrolledWindow:
        sensors = dev.setdefault("sensors", {})
        weather = sensors.setdefault("weather", {})
        g = self._group("Clima")
        self._r_w_en = self._switch(weather, "enabled", "Habilitar clima")
        self._r_w_city = self._entry(weather, "city", "Cidade")
        self._r_w_city.set_text(str(weather.get("city", "São Paulo,BR")))
        self._r_w_int = self._entry(weather, "interval", "Intervalo")
        self._r_w_int.set_text(str(weather.get("interval", "30m")))
        for r in (self._r_w_en, self._r_w_city, self._r_w_int):
            g.add(r)
        return self._scrolled(g)

    # ------------------------------------------------------------------
    # Salvar
    # ------------------------------------------------------------------

    def _on_save(self, _btn):
        dev = self._cfg.setdefault("device", {})
        disp = dev.setdefault("display", {})
        sensors = dev.setdefault("sensors", {})
        weather = sensors.setdefault("weather", {})

        dev["port"] = self._r_port.get_text()
        dev["api_port"] = int(self._r_api.get_value())
        dev["log"] = self.LOG_LEVELS[self._r_log.get_selected()]
        dev["turn_off_on_exit"] = self._r_tur.get_active()

        disp["brightness"] = int(self._r_bright.get_value())
        disp["width"] = int(self._r_w.get_value())
        disp["height"] = int(self._r_h.get_value())
        disp["reverse"] = self._r_rev.get_active()

        sensors.setdefault("cpu", {})["interval"] = self._r_cpu_int.get_text()
        sensors["cpu"]["temperature_sensor"] = self._r_cpu_temp.get_text()
        sensors.setdefault("gpu", {})["interval"] = self._r_gpu_int.get_text()
        sensors["gpu"]["provider"] = self.GPU_PROVIDERS[self._r_gpu_prov.get_selected()]
        sensors.setdefault("memory", {})["interval"] = self._r_mem_int.get_text()
        sensors.setdefault("disk", {})["interval"] = self._r_disk_int.get_text()
        sensors["disk"]["temperature_sensor"] = self._r_disk_temp.get_text()
        sensors.setdefault("network", {})["eth"] = self._r_eth.get_text()
        sensors["network"]["wlo"] = self._r_wlo.get_text()
        sensors["network"]["interval"] = self._r_net_int.get_text()
        weather["enabled"] = self._r_w_en.get_active()
        weather["city"] = self._r_w_city.get_text()
        weather["interval"] = self._r_w_int.get_text()

        if self._save_yaml():
            self.close()
            self._info("Configurações", "Salvo. Reinicie o daemon para aplicar as alterações.")

    def _info(self, heading, body):
        dlg = Adw.AlertDialog()
        dlg.set_heading(heading)
        dlg.set_body(body)
        dlg.add_response("ok", "OK")
        dlg.present(self)


def _get_path(d: dict, path: str):
    """Acessa d[a][b][c] a partir de 'a.b.c'."""
    cur = d
    for part in path.split("."):
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def show_config_dialog(parent):
    """Cria e mostra a janela de configurações."""
    ConfigDialog(parent).present()
