import subprocess
import threading
import logging
from typing import Optional

from gi.repository import GLib, GdkPixbuf, Gtk, Gdk

log = logging.getLogger(__name__)

VIDEO_FPS = 24


class VideoPlayer:
    """
    Streams video via ffmpeg subprocess → raw RGBA → GtkPicture.
    Same approach as the Go version: decode to raw pixels in a thread,
    push to GTK via GLib.idle_add.
    """

    def __init__(self, frame_w: int = 1280, frame_h: int = 720):
        self._frame_w = frame_w
        self._frame_h = frame_h
        self._proc: Optional[subprocess.Popen] = None
        self._thread: Optional[threading.Thread] = None
        self._stop_event = threading.Event()
        self._picture = Gtk.Picture()
        self._picture.set_can_shrink(False)
        self._picture.set_content_fit(Gtk.ContentFit.FILL)

    @property
    def widget(self) -> Gtk.Picture:
        return self._picture

    def set_size(self, w: int, h: int):
        self._frame_w = w
        self._frame_h = h

    def load(self, path: str):
        self.stop()
        self._stop_event.clear()
        self._thread = threading.Thread(
            target=self._stream_loop, args=(path,), daemon=True
        )
        self._thread.start()

    def stop(self):
        self._stop_event.set()
        if self._proc:
            try:
                self._proc.kill()
            except Exception:
                pass
            self._proc = None
        if self._thread:
            self._thread.join(timeout=2)
            self._thread = None
        GLib.idle_add(self._picture.set_paintable, None)

    def _stream_loop(self, path: str):
        while not self._stop_event.is_set():
            stopped = self._stream_once(path)
            if stopped:
                return

    @staticmethod
    def _probe_dims(path: str, input_flags: list) -> tuple[int, int]:
        """Return (width, height) of the video stream, or (0, 0) on failure."""
        try:
            r = subprocess.run(
                ["ffprobe", "-v", "error",
                 *input_flags, "-i", path,
                 "-select_streams", "v:0",
                 "-show_entries", "stream=width,height",
                 "-of", "csv=p=0"],
                capture_output=True, text=True, timeout=5,
            )
            parts = r.stdout.strip().split(",")
            if len(parts) == 2:
                return int(parts[0]), int(parts[1])
        except Exception:
            pass
        return 0, 0

    def _stream_once(self, path: str) -> bool:
        w, h = self._frame_w, self._frame_h
        frame_bytes = w * h * 4

        input_flags = ["-f", "h264"] if path.lower().endswith(".h264") else []

        # Decide rotation: only rotate if the video's own orientation doesn't
        # match the canvas orientation (portrait video → landscape canvas, etc.).
        vf_parts = []
        vw, vh = self._probe_dims(path, input_flags)
        if vw > 0 and vh > 0:
            video_landscape = vw > vh
            canvas_landscape = w > h
            if video_landscape != canvas_landscape:
                vf_parts.append("transpose=2")
        vf_parts.append(f"fps={VIDEO_FPS},scale={w}:{h}")
        cmd = [
            "ffmpeg", "-hide_banner", "-loglevel", "error",
            *input_flags, "-i", path,
            "-vf", ",".join(vf_parts),
            "-f", "rawvideo", "-pix_fmt", "rgba", "-",
        ]
        try:
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
            self._proc = proc
        except FileNotFoundError:
            log.error("ffmpeg not found")
            return True

        import time
        frame_dur = 1.0 / VIDEO_FPS

        while not self._stop_event.is_set():
            t0 = time.monotonic()
            raw = proc.stdout.read(frame_bytes)
            if len(raw) < frame_bytes:
                break

            frame = bytes(raw)  # copy before passing to idle
            GLib.idle_add(self._update_frame, frame, w, h)

            elapsed = time.monotonic() - t0
            remaining = frame_dur - elapsed
            if remaining > 0:
                self._stop_event.wait(timeout=remaining)

        proc.stdout.close()
        proc.wait()
        self._proc = None
        return self._stop_event.is_set()

    def _update_frame(self, raw: bytes, w: int, h: int):
        try:
            pixbuf = GdkPixbuf.Pixbuf.new_from_bytes(
                GLib.Bytes.new(raw),
                GdkPixbuf.Colorspace.RGB,
                True,   # has_alpha
                8,      # bits_per_sample
                w, h,
                w * 4,  # rowstride
            )
            texture = Gdk.Texture.new_for_pixbuf(pixbuf)
            self._picture.set_paintable(texture)
        except Exception as e:
            log.debug("frame update error: %s", e)
