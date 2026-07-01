from __future__ import annotations
import dataclasses
from dataclasses import dataclass, field
from typing import Optional, Dict, get_type_hints, get_origin, get_args, Union
import yaml


# ---------------------------------------------------------------------------
# Generic from_dict / to_dict helpers
# ---------------------------------------------------------------------------

def _from_dict(cls, data):
    """Recursively build a dataclass from a plain dict, ignoring unknown keys.

    Also records which fields were present in the source (``_set_fields``) so
    that serialization can reproduce the original structure instead of emitting
    every dataclass default as noise.
    """
    if data is None:
        return None
    if not isinstance(data, dict):
        return data

    hints = get_type_hints(cls)
    kwargs = {}
    for f in dataclasses.fields(cls):
        if f.name not in data:
            continue
        val = data[f.name]
        hint = hints.get(f.name)
        if hint is None:
            kwargs[f.name] = val
            continue

        origin = get_origin(hint)
        args = get_args(hint)

        if origin is Union:
            # Optional[X] → Union[X, None]
            inner = next((a for a in args if a is not type(None)), None)
            if inner and dataclasses.is_dataclass(inner) and isinstance(val, dict):
                val = _from_dict(inner, val)
        elif origin is dict and len(args) == 2 and dataclasses.is_dataclass(args[1]):
            val = {k: _from_dict(args[1], v) for k, v in val.items()} if val else {}
        elif dataclasses.is_dataclass(hint) and isinstance(val, dict):
            val = _from_dict(hint, val)

        kwargs[f.name] = val

    obj = cls(**kwargs)
    obj._set_fields = set(kwargs.keys())
    return obj


def _is_default_value(f, val):
    """True when ``val`` equals the field's declared default (or factory output)."""
    if f.default is not dataclasses.MISSING:
        return val == f.default
    if f.default_factory is not dataclasses.MISSING:
        try:
            return val == f.default_factory()
        except Exception:
            return False
    return False


# Enhancement fields whose zero/false value is pure noise ("off" by default) —
# never worth writing for a freshly created element.  Does NOT include fields
# like RADIUS/MIN_VALUE/SHOW_TEXT whose default is still a meaningful value.
_NOISE_ZERO_INT = frozenset({
    "BLOCK_ANGLE", "BORDER_WIDTH", "CORNER_RADIUS",
    "STEPS", "STEP_GAP", "BLOCK_WIDTH", "INDEX",
})
_NOISE_FALSE_BOOL = frozenset({"ROUND", "REVERT", "REVERT_VALUE"})


def _is_noise_default(f, val):
    if isinstance(val, bool):
        return (not val) and f.name in _NOISE_FALSE_BOOL
    if isinstance(val, int) and val == 0:
        return f.name in _NOISE_ZERO_INT
    return False


def _to_dict(obj):
    """Recursively convert a dataclass to a plain dict.

    Two regimes:

    * **Loaded from source** (``_set_fields`` set): reproduce the original
      field set exactly — emit every source field, plus any field the editor
      moved off its default; drop defaults that were never in the source.  This
      is what lets a round-trip match the original file (even quirks like
      SHOW_TEXT:false present in one block but absent in another) and keeps a
      single edit from rewriting unrelated fields.

    * **Newly created** (no ``_set_fields``): keep every meaningful field even
      when it equals the dataclass default (RADIUS:80 on a freshly added dial
      must survive), and only drop pure-noise enhancement defaults.
    """
    if obj is None:
        return None
    if dataclasses.is_dataclass(obj):
        set_fields = getattr(obj, "_set_fields", None)
        result = {}
        for f in dataclasses.fields(obj):
            val = getattr(obj, f.name)
            if val is None:
                continue
            # Empty dicts / strings are never meaningful theme values.
            if isinstance(val, (dict, str)) and not val:
                continue
            if set_fields is not None:
                if f.name not in set_fields and _is_default_value(f, val):
                    continue
            else:
                if _is_noise_default(f, val):
                    continue
            serialized = _to_dict(val)
            if serialized is not None:
                result[f.name] = serialized
        return result or None
    if isinstance(obj, dict):
        out = {}
        for k, v in obj.items():
            s = _to_dict(v)
            if s is not None:
                out[k] = s
        return out or None
    if isinstance(obj, list):
        return [_to_dict(v) for v in obj] or None
    return obj


# ---------------------------------------------------------------------------
# Leaf types — field order matches expected YAML output order
# ---------------------------------------------------------------------------

@dataclass
class Text:
    SHOW: bool = True
    INDEX: int = 0
    X: int = 0
    Y: int = 0
    FONT: str = ""
    FONT_SIZE: int = 16
    FONT_COLOR: str = ""
    SHOW_UNIT: bool = False
    PLACEHOLDER: str = ""
    TEXT: str = ""
    ALIGN: str = ""
    FORMAT: str = ""
    BACKGROUND_COLOR: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Graph:
    SHOW: bool = True
    INDEX: int = 0
    X: int = 0
    Y: int = 0
    WIDTH: int = 200
    HEIGHT: int = 100
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    BAR_COLOR: str = "#00FF00"
    BAR_OUTLINE: bool = False
    BACKGROUND_COLOR: str = ""
    GRADIENT_COLOR: str = ""
    BORDER_WIDTH: int = 0
    CORNER_RADIUS: int = 0
    STEPS: int = 0
    STEP_GAP: int = 0
    BLOCK_WIDTH: int = 0
    REVERT_VALUE: bool = False
    DIRECTION: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Radial:
    SHOW: bool = True
    INDEX: int = 0
    X: int = 0
    Y: int = 0
    RADIUS: int = 80
    WIDTH: int = 10
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    ANGLE_START: int = -210
    ANGLE_END: int = 30
    ANGLE_STEPS: int = 0
    ANGLE_SEP: int = 0
    CLOCKWISE: bool = True
    BAR_COLOR: str = "#00FF00"
    BACKGROUND_COLOR: str = ""
    GRADIENT_COLOR: str = ""
    BLOCK_ANGLE: int = 0
    ROUND: bool = False
    REVERT: bool = False
    REVERT_VALUE: bool = False
    SHOW_TEXT: bool = False
    SHOW_UNIT: bool = False
    FONT: str = ""
    FONT_COLOR: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Chart:
    SHOW: bool = True
    INDEX: int = 0
    STYLE: str = ""
    X: int = 0
    Y: int = 0
    WIDTH: int = 200
    HEIGHT: int = 100
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    COLUMN_WIDTH: int = 4
    COLUMN_GAP: int = 1
    FILL_COLOR: str = "#00FF00"
    LINE_COLOR: str = "#00FF00"
    BORDER_WIDTH: int = 0
    MAX_SAMPLES: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Gauge:
    SHOW: bool = True
    INDEX: int = 0
    X: int = 0
    Y: int = 0
    RADIUS: int = 80
    NEEDLE_WIDTH: int = 4
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    ANGLE_START: int = -210
    ANGLE_END: int = 30
    NEEDLE_COLOR: str = "#FF0000"
    SHOW_TEXT: bool = False
    SHOW_UNIT: bool = False
    FONT: str = ""
    FONT_COLOR: str = ""
    BACKGROUND_COLOR: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class StatusBar:
    SHOW: bool = True
    INDEX: int = 0
    X: int = 0
    Y: int = 0
    WIDTH: int = 200
    HEIGHT: int = 20
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    BAR_COLOR: str = "#00FF00"
    INDICATOR_COLOR: str = "#FFFFFF"
    INDICATOR_RADIUS: int = 0
    BACKGROUND_COLOR: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class StaticImage:
    PATH: str = ""
    X: int = 0
    Y: int = 0
    WIDTH: int = 0
    HEIGHT: int = 0
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class DinamicImage:
    PATH: str = ""
    X: int = 0
    Y: int = 0
    WIDTH: int = 0
    HEIGHT: int = 0
    BACKGROUND: str = ""

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


# ---------------------------------------------------------------------------
# Measurement container (used by CPU, GPU, Disk, DateTime, Weather…)
# ---------------------------------------------------------------------------

@dataclass
class Measurement:
    SHOW: bool = True
    INDEX: int = 0
    Graph: Optional[Graph] = None
    Radial: Optional[Radial] = None
    Gauge: Optional[Gauge] = None
    StatusBar: Optional[StatusBar] = None
    Chart: Optional[Chart] = None
    Text: Optional[Text] = None
    PercentText: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            SHOW=d.get("SHOW", True),
            INDEX=d.get("INDEX", 0),
            Graph=Graph.from_dict(d.get("GRAPH")),
            Radial=Radial.from_dict(d.get("RADIAL")),
            Gauge=Gauge.from_dict(d.get("GAUGE")),
            StatusBar=StatusBar.from_dict(d.get("STATUS_BAR")),
            Chart=Chart.from_dict(d.get("CHART")),
            Text=Text.from_dict(d.get("TEXT")),
            PercentText=Text.from_dict(d.get("PERCENT_TEXT")),
        )

    def to_dict(self):
        d = {}
        # Do NOT write SHOW — Go's Mesurement struct has no SHOW field.
        if self.INDEX:
            d["INDEX"] = self.INDEX
        # Radial before Graph to match conventional YAML order.
        if self.Radial:
            d["RADIAL"] = _to_dict(self.Radial)
        if self.Graph:
            d["GRAPH"] = _to_dict(self.Graph)
        if self.Gauge:
            d["GAUGE"] = _to_dict(self.Gauge)
        if self.StatusBar:
            d["STATUS_BAR"] = _to_dict(self.StatusBar)
        if self.Chart:
            d["CHART"] = _to_dict(self.Chart)
        if self.Text:
            d["TEXT"] = _to_dict(self.Text)
        if self.PercentText:
            d["PERCENT_TEXT"] = _to_dict(self.PercentText)
        return d


# ---------------------------------------------------------------------------
# Stats sub-types
# ---------------------------------------------------------------------------

@dataclass
class Load:
    One: Optional[Measurement] = None
    Five: Optional[Measurement] = None
    Fifteen: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            One=Measurement.from_dict(d.get("ONE")),
            Five=Measurement.from_dict(d.get("FIVE")),
            Fifteen=Measurement.from_dict(d.get("FIFTEEN")),
        )


@dataclass
class CPU:
    Model: Optional[Measurement] = None
    Percentage: Optional[Measurement] = None
    Temperature: Optional[Measurement] = None
    Frequency: Optional[Measurement] = None
    Load: Optional[Load] = None
    Fan: Optional[Measurement] = None
    Power: Optional[Measurement] = None
    Voltage: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Model=Measurement.from_dict(d.get("MODEL")),
            Percentage=Measurement.from_dict(d.get("PERCENTAGE")),
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
            Frequency=Measurement.from_dict(d.get("FREQUENCY")),
            Load=Load.from_dict(d.get("LOAD")),
            Fan=Measurement.from_dict(d.get("FAN")),
            Power=Measurement.from_dict(d.get("POWER")),
            Voltage=Measurement.from_dict(d.get("VOLTAGE")),
        )


@dataclass
class GPU:
    Model: Optional[Measurement] = None
    Frequency: Optional[Measurement] = None
    Temperature: Optional[Measurement] = None
    Percentage: Optional[Measurement] = None
    Power: Optional[Measurement] = None
    Memory: Optional[Measurement] = None
    Voltage: Optional[Measurement] = None
    Fan: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Model=Measurement.from_dict(d.get("MODEL")),
            Frequency=Measurement.from_dict(d.get("FREQUENCY")),
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
            Percentage=Measurement.from_dict(d.get("PERCENTAGE")),
            Power=Measurement.from_dict(d.get("POWER")),
            Memory=Measurement.from_dict(d.get("MEMORY")),
            Voltage=Measurement.from_dict(d.get("VOLTAGE")),
            Fan=Measurement.from_dict(d.get("FAN")),
        )


@dataclass
class Memory:
    Model: Optional[Measurement] = None
    Total: Optional[Measurement] = None
    Swap: Optional[Measurement] = None
    Virtual: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Model=Measurement.from_dict(d.get("MODEL")),
            Total=Measurement.from_dict(d.get("TOTAL")),
            Swap=Measurement.from_dict(d.get("SWAP")),
            Virtual=Measurement.from_dict(d.get("VIRTUAL")),
        )


@dataclass
class Disk:
    Used: Optional[Measurement] = None
    Total: Optional[Measurement] = None
    Free: Optional[Measurement] = None
    Temperature: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Used=Measurement.from_dict(d.get("USED")),
            Total=Measurement.from_dict(d.get("TOTAL")),
            Free=Measurement.from_dict(d.get("FREE")),
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
        )


@dataclass
class NetworkMeasurement:
    Upload: Optional[Measurement] = None
    Download: Optional[Measurement] = None
    Uploaded: Optional[Measurement] = None
    Downloaded: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Upload=Measurement.from_dict(d.get("UPLOAD")),
            Download=Measurement.from_dict(d.get("DOWNLOAD")),
            Uploaded=Measurement.from_dict(d.get("UPLOADED")),
            Downloaded=Measurement.from_dict(d.get("DOWNLOADED")),
        )


@dataclass
class Network:
    Wifi: Optional[NetworkMeasurement] = None
    Wired: Optional[NetworkMeasurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Wifi=NetworkMeasurement.from_dict(d.get("WLO")),
            Wired=NetworkMeasurement.from_dict(d.get("ETH")),
        )


@dataclass
class DateTime:
    Day: Optional[Measurement] = None
    Hour: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Day=Measurement.from_dict(d.get("DAY")),
            Hour=Measurement.from_dict(d.get("HOUR")),
        )


@dataclass
class Weather:
    Temperature: Optional[Measurement] = None
    Condition: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
            Condition=Text.from_dict(d.get("CONDITION")),
        )




@dataclass
class Stats:
    CPU: Optional[CPU] = None
    GPU: Optional[GPU] = None
    Memory: Optional[Memory] = None
    Disk: Optional[Disk] = None
    Net: Optional[Network] = None
    Date: Optional[DateTime] = None
    Weather: Optional[Weather] = None
    Volume: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            CPU=CPU.from_dict(d.get("CPU")),
            GPU=GPU.from_dict(d.get("GPU")),
            Memory=Memory.from_dict(d.get("MEMORY")),
            Disk=Disk.from_dict(d.get("DISK")),
            Net=Network.from_dict(d.get("NET")),
            Date=DateTime.from_dict(d.get("DATE")),
            Weather=Weather.from_dict(d.get("WEATHER")),
            Volume=Measurement.from_dict(d.get("VOLUME")),
        )


@dataclass
class Display:
    SIZE: str = "TURZX"
    ORIENTATION: str = "landscape"
    RGB_LED: str = ""
    WIDTH: int = 1280
    HEIGHT: int = 720

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d) if d else cls()


# ---------------------------------------------------------------------------
# Root Theme
# ---------------------------------------------------------------------------

@dataclass
class Theme:
    display: Optional[Display] = None
    static_images: Dict[str, StaticImage] = field(default_factory=dict)
    video: Optional[DinamicImage] = None
    static_texts: Dict[str, Text] = field(default_factory=dict)
    STATS: Optional[Stats] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return cls()
        static_images = {}
        for k, v in (d.get("static_images") or {}).items():
            static_images[k] = StaticImage.from_dict(v)
        static_texts = {}
        for k, v in (d.get("static_texts") or {}).items():
            static_texts[k] = Text.from_dict(v)
        return cls(
            display=Display.from_dict(d.get("display")),
            static_images=static_images,
            video=DinamicImage.from_dict(d.get("video")),
            static_texts=static_texts,
            STATS=Stats.from_dict(d.get("STATS")),
        )

    def to_dict(self):
        d = {}
        if self.display:
            dd = _to_dict(self.display)
            if dd:
                d["display"] = dd
        if self.static_images:
            d["static_images"] = {k: _to_dict(v) for k, v in self.static_images.items()}
        if self.video:
            dv = _to_dict(self.video)
            if dv:
                d["video"] = dv
        if self.static_texts:
            d["static_texts"] = {k: _to_dict(v) for k, v in self.static_texts.items()}
        if self.STATS:
            ds = self._stats_to_dict()
            if ds:
                d["STATS"] = ds
        return d

    def _stats_to_dict(self):
        s = self.STATS
        if not s:
            return None
        out = {}
        if s.CPU:
            out["CPU"] = self._cpu_to_dict(s.CPU)
        if s.GPU:
            out["GPU"] = self._gpu_to_dict(s.GPU)
        if s.Memory:
            out["MEMORY"] = self._memory_to_dict(s.Memory)
        if s.Disk:
            out["DISK"] = self._disk_to_dict(s.Disk)
        if s.Net:
            out["NET"] = self._net_to_dict(s.Net)
        if s.Date:
            out["DATE"] = self._date_to_dict(s.Date)
        if s.Weather:
            out["WEATHER"] = self._weather_to_dict(s.Weather)
        if s.Volume:
            out["VOLUME"] = self._volume_to_dict(s.Volume)
        return out

    def _measurement_to_dict(self, m):
        if not m:
            return None
        return m.to_dict()

    def _cpu_to_dict(self, cpu):
        d = {}
        # Order matches the conventional theme YAML layout.
        for attr, key in [
            ("Model",       "MODEL"),
            ("Percentage",  "PERCENTAGE"),
            ("Temperature", "TEMPERATURE"),
            ("Frequency",   "FREQUENCY"),
            ("Load",        None),
            ("Fan",         "FAN"),
            ("Power",       "POWER"),
            ("Voltage",     "VOLTAGE"),
        ]:
            if attr == "Load":
                if cpu.Load:
                    ld = {}
                    for slot, key in [("One","ONE"),("Five","FIVE"),("Fifteen","FIFTEEN")]:
                        m = getattr(cpu.Load, slot)
                        if m:
                            ld[key] = self._measurement_to_dict(m)
                    if ld:
                        d["LOAD"] = ld
            else:
                m = getattr(cpu, attr)
                if m:
                    d[key] = self._measurement_to_dict(m)
        return d

    def _gpu_to_dict(self, gpu):
        d = {}
        # Order matches the conventional theme YAML layout.
        for attr, key in [
            ("Model",       "MODEL"),
            ("Frequency",   "FREQUENCY"),
            ("Temperature", "TEMPERATURE"),
            ("Percentage",  "PERCENTAGE"),
            ("Power",       "POWER"),
            ("Memory",      "MEMORY"),
            ("Voltage",     "VOLTAGE"),
            ("Fan",         "FAN"),
        ]:
            m = getattr(gpu, attr)
            if m:
                d[key] = self._measurement_to_dict(m)
        return d

    def _memory_to_dict(self, mem):
        d = {}
        if mem.Model:
            d["MODEL"] = self._measurement_to_dict(mem.Model)
        if mem.Total:
            d["TOTAL"] = self._measurement_to_dict(mem.Total)
        if mem.Virtual:
            d["VIRTUAL"] = self._measurement_to_dict(mem.Virtual)
        if mem.Swap:
            d["SWAP"] = self._measurement_to_dict(mem.Swap)
        return d

    def _disk_to_dict(self, disk):
        d = {}
        for attr, key in [("Used", "USED"), ("Total", "TOTAL"),
                           ("Free", "FREE"), ("Temperature", "TEMPERATURE")]:
            m = getattr(disk, attr)
            if m:
                d[key] = self._measurement_to_dict(m)
        return d

    def _net_to_dict(self, net):
        d = {}
        if net.Wifi:
            nd = {}
            for attr, key in [("Upload","UPLOAD"),("Download","DOWNLOAD"),
                               ("Uploaded","UPLOADED"),("Downloaded","DOWNLOADED")]:
                m = getattr(net.Wifi, attr)
                if m:
                    nd[key] = self._measurement_to_dict(m)
            if nd:
                d["WLO"] = nd
        if net.Wired:
            nd = {}
            for attr, key in [("Upload","UPLOAD"),("Download","DOWNLOAD"),
                               ("Uploaded","UPLOADED"),("Downloaded","DOWNLOADED")]:
                m = getattr(net.Wired, attr)
                if m:
                    nd[key] = self._measurement_to_dict(m)
            if nd:
                d["ETH"] = nd
        return d

    def _date_to_dict(self, dt):
        d = {}
        if dt.Day:
            d["DAY"] = self._measurement_to_dict(dt.Day)
        if dt.Hour:
            d["HOUR"] = self._measurement_to_dict(dt.Hour)
        return d

    def _weather_to_dict(self, w):
        d = {}
        if w.Temperature:
            d["TEMPERATURE"] = self._measurement_to_dict(w.Temperature)
        if w.Condition:
            d["CONDITION"] = _to_dict(w.Condition)
        return d

    def _volume_to_dict(self, v):
        return self._measurement_to_dict(v) or {}

    @classmethod
    def load(cls, path: str) -> "Theme":
        with open(path, "r", encoding="utf-8") as f:
            data = yaml.safe_load(f)
        return cls.from_dict(data or {})

    def save(self, path: str):
        data = self.to_dict()
        with open(path, "w", encoding="utf-8") as f:
            yaml.dump(data, f, allow_unicode=True, sort_keys=False, default_flow_style=False)
