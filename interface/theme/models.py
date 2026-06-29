from __future__ import annotations
import dataclasses
from dataclasses import dataclass, field
from typing import Optional, Dict, get_type_hints, get_origin, get_args, Union
import yaml


# ---------------------------------------------------------------------------
# Generic from_dict / to_dict helpers
# ---------------------------------------------------------------------------

def _from_dict(cls, data):
    """Recursively build a dataclass from a plain dict, ignoring unknown keys."""
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

    return cls(**kwargs)


def _to_dict(obj):
    """Recursively convert a dataclass to a plain dict, omitting None / empty."""
    if obj is None:
        return None
    if dataclasses.is_dataclass(obj):
        result = {}
        for f in dataclasses.fields(obj):
            val = getattr(obj, f.name)
            if val is None:
                continue
            if isinstance(val, dict) and not val:
                continue
            if isinstance(val, bool):
                result[f.name] = val
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
# Leaf types
# ---------------------------------------------------------------------------

@dataclass
class Text:
    X: int = 0
    Y: int = 0
    SHOW: bool = True
    SHOW_UNIT: bool = False
    TEXT: str = ""
    PLACEHOLDER: str = ""
    FONT: str = ""
    FONT_SIZE: int = 16
    FONT_COLOR: str = "#FFFFFF"
    BACKGROUND_COLOR: str = ""
    ALIGN: str = "LEFT"
    FORMAT: str = ""
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Graph:
    SHOW: bool = True
    X: int = 0
    Y: int = 0
    WIDTH: int = 200
    HEIGHT: int = 100
    MIN_VALUE: int = 0
    MAX_VALUE: int = 100
    BAR_COLOR: str = "#00FF00"
    BACKGROUND_COLOR: str = ""
    GRADIENT_COLOR: str = ""
    BAR_OUTLINE: bool = False
    BORDER_WIDTH: int = 0
    CORNER_RADIUS: int = 0
    STEPS: int = 0
    STEP_GAP: int = 0
    BLOCK_WIDTH: int = 0
    REVERT_VALUE: bool = False
    DIRECTION: str = ""
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Radial:
    SHOW: bool = True
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
    BLOCK_ANGLE: int = 0
    CLOCKWISE: bool = True
    BAR_COLOR: str = "#00FF00"
    BACKGROUND_COLOR: str = ""
    GRADIENT_COLOR: str = ""
    ROUND: bool = False
    REVERT: bool = False
    REVERT_VALUE: bool = False
    SHOW_TEXT: bool = False
    SHOW_UNIT: bool = False
    FONT: str = ""
    FONT_COLOR: str = "#FFFFFF"
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Chart:
    SHOW: bool = True
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
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class Gauge:
    SHOW: bool = True
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
    FONT_COLOR: str = "#FFFFFF"
    BACKGROUND_COLOR: str = ""
    INDEX: int = 0

    @classmethod
    def from_dict(cls, d):
        return _from_dict(cls, d)


@dataclass
class StatusBar:
    SHOW: bool = True
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
    INDEX: int = 0

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

    # yaml keys differ from field names for Graph/Radial/etc:
    # YAML uses GRAPH, RADIAL, GAUGE, STATUS_BAR, CHART, TEXT, PERCENT_TEXT
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
        d = {"SHOW": self.SHOW}
        if self.INDEX:
            d["INDEX"] = self.INDEX
        if self.Graph:
            d["GRAPH"] = _to_dict(self.Graph)
        if self.Radial:
            d["RADIAL"] = _to_dict(self.Radial)
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
# Memory uses MemMeasurement (different sub-fields)
# ---------------------------------------------------------------------------

@dataclass
class MemMeasurement:
    SHOW: bool = True
    Graph: Optional[Graph] = None
    Radial: Optional[Radial] = None
    Chart: Optional[Chart] = None
    Used: Optional[Text] = None
    Free: Optional[Text] = None
    PercentText: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            SHOW=d.get("SHOW", True),
            Graph=Graph.from_dict(d.get("GRAPH")),
            Radial=Radial.from_dict(d.get("RADIAL")),
            Chart=Chart.from_dict(d.get("CHART")),
            Used=Text.from_dict(d.get("USED")),
            Free=Text.from_dict(d.get("FREE")),
            PercentText=Text.from_dict(d.get("PERCENT_TEXT")),
        )

    def to_dict(self):
        d = {"SHOW": self.SHOW}
        if self.Graph:
            d["GRAPH"] = _to_dict(self.Graph)
        if self.Radial:
            d["RADIAL"] = _to_dict(self.Radial)
        if self.Chart:
            d["CHART"] = _to_dict(self.Chart)
        if self.Used:
            d["USED"] = _to_dict(self.Used)
        if self.Free:
            d["FREE"] = _to_dict(self.Free)
        if self.PercentText:
            d["PERCENT_TEXT"] = _to_dict(self.PercentText)
        return d


# ---------------------------------------------------------------------------
# Stats sub-types
# ---------------------------------------------------------------------------

@dataclass
class LoadOne:
    Text: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        return cls(Text=Text.from_dict(d.get("TEXT"))) if d else None


@dataclass
class LoadFive:
    Text: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        return cls(Text=Text.from_dict(d.get("TEXT"))) if d else None


@dataclass
class LoadFifteen:
    Text: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        return cls(Text=Text.from_dict(d.get("TEXT"))) if d else None


@dataclass
class Load:
    One: Optional[LoadOne] = None
    Five: Optional[LoadFive] = None
    Fifteen: Optional[LoadFifteen] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            One=LoadOne.from_dict(d.get("ONE")),
            Five=LoadFive.from_dict(d.get("FIVE")),
            Fifteen=LoadFifteen.from_dict(d.get("FIFTEEN")),
        )


@dataclass
class CPU:
    Percentage: Optional[Measurement] = None
    Frequency: Optional[Measurement] = None
    Load: Optional[Load] = None
    Temperature: Optional[Measurement] = None
    Fan: Optional[Measurement] = None
    Power: Optional[Measurement] = None
    Voltage: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Percentage=Measurement.from_dict(d.get("PERCENTAGE")),
            Frequency=Measurement.from_dict(d.get("FREQUENCY")),
            Load=Load.from_dict(d.get("LOAD")),
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
            Fan=Measurement.from_dict(d.get("FAN")),
            Power=Measurement.from_dict(d.get("POWER")),
            Voltage=Measurement.from_dict(d.get("VOLTAGE")),
        )


@dataclass
class GPU:
    Percentage: Optional[Measurement] = None
    Memory: Optional[Measurement] = None
    Temperature: Optional[Measurement] = None
    Power: Optional[Measurement] = None
    Frequency: Optional[Measurement] = None
    Voltage: Optional[Measurement] = None
    Fan: Optional[Measurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Percentage=Measurement.from_dict(d.get("PERCENTAGE")),
            Memory=Measurement.from_dict(d.get("MEMORY")),
            Temperature=Measurement.from_dict(d.get("TEMPERATURE")),
            Power=Measurement.from_dict(d.get("POWER")),
            Frequency=Measurement.from_dict(d.get("FREQUENCY")),
            Voltage=Measurement.from_dict(d.get("VOLTAGE")),
            Fan=Measurement.from_dict(d.get("FAN")),
        )


@dataclass
class Memory:
    Swap: Optional[MemMeasurement] = None
    Virtual: Optional[MemMeasurement] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(
            Swap=MemMeasurement.from_dict(d.get("SWAP")),
            Virtual=MemMeasurement.from_dict(d.get("VIRTUAL")),
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
class Volume:
    Text: Optional[Text] = None

    @classmethod
    def from_dict(cls, d):
        if not d:
            return None
        return cls(Text=Text.from_dict(d.get("TEXT")))


@dataclass
class Stats:
    CPU: Optional[CPU] = None
    GPU: Optional[GPU] = None
    Memory: Optional[Memory] = None
    Disk: Optional[Disk] = None
    Net: Optional[Network] = None
    Date: Optional[DateTime] = None
    Weather: Optional[Weather] = None
    Volume: Optional[Volume] = None

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
            Volume=Volume.from_dict(d.get("VOLUME")),
        )


@dataclass
class Display:
    SIZE: str = "TURZX"
    ORIENTATION: str = "landscape"
    WIDTH: int = 1280
    HEIGHT: int = 720
    RGB_LED: str = ""

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

    def _mem_to_dict(self, m):
        if not m:
            return None
        return m.to_dict()

    def _cpu_to_dict(self, cpu):
        d = {}
        if cpu.Percentage:
            d["PERCENTAGE"] = self._measurement_to_dict(cpu.Percentage)
        if cpu.Frequency:
            d["FREQUENCY"] = self._measurement_to_dict(cpu.Frequency)
        if cpu.Load:
            ld = {}
            if cpu.Load.One and cpu.Load.One.Text:
                ld["ONE"] = {"TEXT": _to_dict(cpu.Load.One.Text)}
            if cpu.Load.Five and cpu.Load.Five.Text:
                ld["FIVE"] = {"TEXT": _to_dict(cpu.Load.Five.Text)}
            if cpu.Load.Fifteen and cpu.Load.Fifteen.Text:
                ld["FIFTEEN"] = {"TEXT": _to_dict(cpu.Load.Fifteen.Text)}
            if ld:
                d["LOAD"] = ld
        if cpu.Temperature:
            d["TEMPERATURE"] = self._measurement_to_dict(cpu.Temperature)
        if cpu.Fan:
            d["FAN"] = self._measurement_to_dict(cpu.Fan)
        if cpu.Power:
            d["POWER"] = self._measurement_to_dict(cpu.Power)
        if cpu.Voltage:
            d["VOLTAGE"] = self._measurement_to_dict(cpu.Voltage)
        return d

    def _gpu_to_dict(self, gpu):
        d = {}
        for attr, key in [("Percentage", "PERCENTAGE"), ("Memory", "MEMORY"),
                           ("Temperature", "TEMPERATURE"), ("Power", "POWER"),
                           ("Frequency", "FREQUENCY"), ("Voltage", "VOLTAGE"), ("Fan", "FAN")]:
            m = getattr(gpu, attr)
            if m:
                d[key] = self._measurement_to_dict(m)
        return d

    def _memory_to_dict(self, mem):
        d = {}
        if mem.Virtual:
            d["VIRTUAL"] = self._mem_to_dict(mem.Virtual)
        if mem.Swap:
            d["SWAP"] = self._mem_to_dict(mem.Swap)
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
        if v.Text:
            return {"TEXT": _to_dict(v.Text)}
        return {}

    @classmethod
    def load(cls, path: str) -> "Theme":
        with open(path, "r", encoding="utf-8") as f:
            data = yaml.safe_load(f)
        return cls.from_dict(data or {})

    def save(self, path: str):
        data = self.to_dict()
        with open(path, "w", encoding="utf-8") as f:
            yaml.dump(data, f, allow_unicode=True, sort_keys=False, default_flow_style=False)
