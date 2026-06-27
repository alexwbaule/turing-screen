import asyncio
import json
import logging
import threading
import time
from typing import Callable, Optional

import websockets
from gi.repository import GLib

log = logging.getLogger(__name__)

DEFAULT_WS_PORT = 9120


class WSClient:
    """
    WebSocket client that runs an asyncio loop in a background thread.
    Incoming messages are dispatched to GTK's main thread via GLib.idle_add.
    """

    def __init__(self, port: int = DEFAULT_WS_PORT):
        self._url = f"ws://localhost:{port}/ws"
        self._loop: Optional[asyncio.AbstractEventLoop] = None
        self._thread: Optional[threading.Thread] = None
        self._ws = None
        self._connected = False
        self._msg_id = 0
        self._pending: dict[str, asyncio.Future] = {}
        self._pending_lock = threading.Lock()
        self._event_handler: Optional[Callable] = None
        self._running = False

    # ------------------------------------------------------------------
    # Public API (called from GTK main thread)
    # ------------------------------------------------------------------

    def set_event_handler(self, handler: Callable):
        """handler(action: str, payload: dict) called on GTK main thread."""
        self._event_handler = handler

    def start(self):
        """Start background thread with asyncio loop + auto-reconnect."""
        self._running = True
        self._loop = asyncio.new_event_loop()
        self._thread = threading.Thread(target=self._run_loop, daemon=True)
        self._thread.start()

    def stop(self):
        self._running = False
        if self._loop and self._loop.is_running():
            self._loop.call_soon_threadsafe(self._loop.stop)

    def is_connected(self) -> bool:
        return self._connected

    def send(self, action: str, payload: dict = None,
             callback: Callable = None):
        """
        Fire-and-forget send. If callback is provided it receives (response_dict, error_str).
        Both callback and error are delivered on the GTK main thread.
        """
        if not self._connected or not self._loop:
            if callback:
                GLib.idle_add(callback, None, "not connected")
            return

        asyncio.run_coroutine_threadsafe(
            self._send_async(action, payload, callback),
            self._loop,
        )

    # ------------------------------------------------------------------
    # Internals
    # ------------------------------------------------------------------

    def _run_loop(self):
        asyncio.set_event_loop(self._loop)
        self._loop.run_until_complete(self._reconnect_loop())

    async def _reconnect_loop(self):
        while self._running:
            try:
                await self._connect_and_read()
            except Exception as e:
                log.debug("WS connection lost: %s — retry in 3s", e)
            self._connected = False
            self._drain_pending("disconnected")
            if self._running:
                await asyncio.sleep(3)

    async def _connect_and_read(self):
        async with websockets.connect(
            self._url,
            ping_interval=20,
            ping_timeout=10,
            close_timeout=5,
        ) as ws:
            self._ws = ws
            self._connected = True
            log.info("WS connected to %s", self._url)
            GLib.idle_add(self._fire_connection_event, True)

            async for raw in ws:
                try:
                    msg = json.loads(raw)
                except json.JSONDecodeError:
                    continue

                msg_id = msg.get("id", "")
                action = msg.get("action", "")

                with self._pending_lock:
                    future = self._pending.get(msg_id)

                if future and not future.done():
                    future.set_result(msg)
                else:
                    # Push event → dispatch to GTK thread
                    if self._event_handler and action:
                        payload = msg.get("payload") or {}
                        GLib.idle_add(self._event_handler, action, payload)

        self._ws = None
        self._connected = False
        GLib.idle_add(self._fire_connection_event, False)

    async def _send_async(self, action: str, payload: dict, callback: Callable):
        self._msg_id += 1
        msg_id = str(self._msg_id)
        message = {"id": msg_id, "action": action}
        if payload:
            message["payload"] = payload

        future = self._loop.create_future()
        with self._pending_lock:
            self._pending[msg_id] = future

        try:
            await self._ws.send(json.dumps(message))
            response = await asyncio.wait_for(future, timeout=10.0)
            if callback:
                err = response.get("error") if response.get("status") == "error" else None
                GLib.idle_add(callback, response.get("payload"), err)
        except asyncio.TimeoutError:
            if callback:
                GLib.idle_add(callback, None, "timeout")
        except Exception as e:
            if callback:
                GLib.idle_add(callback, None, str(e))
        finally:
            with self._pending_lock:
                self._pending.pop(msg_id, None)

    def _drain_pending(self, reason: str):
        with self._pending_lock:
            for fut in self._pending.values():
                if not fut.done():
                    fut.set_exception(Exception(reason))
            self._pending.clear()

    def _fire_connection_event(self, connected: bool):
        if self._event_handler:
            self._event_handler("connection.changed", {"connected": connected})
