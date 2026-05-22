#!/usr/bin/env python3
"""
Common utilities for Turing Smart Screen 5" (Rev C) scripts.

Protocol validated against PCAPs:
  - NZXTDinamicTheme.pcapng (Linux, play only)
  - BlueTheme-Flip180.pcapng (Linux, upload + play)
  - gundam.pcapng (Windows, upload + play)
  - teste.pcap (Windows, upload + play)
"""

import serial
import serial.tools.list_ports
import sys
import time

DEVICE_SERIAL = "20080411"
DEVICE_SERIAL_SLEEP = "USB7INCH"
BAUD_RATE = 115200
READ_TIMEOUT = 5


def find_device():
    """Find the Turing device serial port. Wakes it up if sleeping."""
    ports = serial.tools.list_ports.comports()
    for port in ports:
        if port.serial_number == DEVICE_SERIAL:
            return port.device
        if port.serial_number == DEVICE_SERIAL_SLEEP:
            print(f"  Device sleeping on {port.device}, waking up...")
            try:
                p = serial.Serial(port.device, BAUD_RATE, timeout=1)
                p.close()
            except Exception:
                pass
            print("  Waiting 20s...")
            time.sleep(20)
            return find_device()
    return None


def pad(cmd, size=250):
    """Pad command to 250 bytes with zeros."""
    return cmd + b'\x00' * (size - len(cmd))


def build_path_cmd(cmd_id, path):
    """Build path-based command: [cmd][0xEF69][path_len:4BE][0x000000][path]"""
    path_bytes = path.encode('ascii')
    payload = bytes([cmd_id, 0xef, 0x69])
    payload += len(path_bytes).to_bytes(4, 'big')
    payload += b'\x00\x00\x00'
    payload += path_bytes
    return payload


def build_create_file(path, file_size):
    """CREATE_FILE: [0x6f][0xEF69][path_len:4BE][0x000000][path][file_size:4LE]"""
    path_bytes = path.encode('ascii')
    payload = bytes([0x6f, 0xef, 0x69])
    payload += len(path_bytes).to_bytes(4, 'big')
    payload += b'\x00\x00\x00'
    payload += path_bytes
    payload += file_size.to_bytes(4, 'little')
    return payload


def build_play_video(path, loop=True):
    """PLAY_VIDEO: [0x78][0xEF69][path_len:4BE][loop:1][0x0000][path]"""
    path_bytes = path.encode('ascii')
    payload = bytes([0x78, 0xef, 0x69])
    payload += len(path_bytes).to_bytes(4, 'big')
    payload += bytes([0x01 if loop else 0x00, 0x00, 0x00])
    payload += path_bytes
    return payload


# Pre-built commands
CMD_HELLO = bytes([0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3])
CMD_STOP_VIDEO = bytes([0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01])
CMD_STOP_MEDIA = bytes([0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01])
CMD_GET_STORAGE = bytes([0x64, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01])
CMD_RESTART = bytes([0x82, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01])


def cmd_brightness(percent):
    """SET_BRIGHTNESS command. percent: 0-100."""
    value = int((percent / 100.0) * 255)
    return bytes([0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, value])


class TuringDevice:
    """High-level interface to the Turing Smart Screen 5"."""

    def __init__(self, port_name=None, brightness=20):
        if port_name is None:
            port_name = find_device()
        if port_name is None:
            print("ERROR: Device not found!")
            sys.exit(1)
        self.port_name = port_name
        self.brightness = brightness
        self.port = serial.Serial(port_name, BAUD_RATE, timeout=READ_TIMEOUT,
                                   dsrdtr=False, rtscts=False)
        self.port.dtr = True
        self.port.rts = True
        self.port.reset_input_buffer()

    def close(self):
        self.port.close()

    def send(self, name, cmd, response_size=0):
        """Send a padded command and optionally read response."""
        self.port.write(pad(cmd))
        if response_size > 0:
            resp = self.port.read(response_size)
            text = resp.rstrip(b'\x00').decode('ascii', errors='replace')
            print(f"  {name} -> \"{text}\"")
            return text
        else:
            print(f"  {name}")
            return None

    def flush_input(self):
        """Flush any leftover data in the serial input buffer."""
        time.sleep(0.3)
        while self.port.in_waiting > 0:
            self.port.read(self.port.in_waiting)
            time.sleep(0.1)
        self.port.reset_input_buffer()

    def init(self):
        """
        Standard device initialization sequence.
        Validated from ALL PCAPs — always the same regardless of video/static.
        Returns the HELLO response string.
        """
        resp = self.send("HELLO", CMD_HELLO, 23)
        if not resp:
            print("ERROR: No HELLO response!")
            sys.exit(1)
        self.send("STOP_VIDEO", CMD_STOP_VIDEO)
        self.send("STOP_MEDIA", CMD_STOP_MEDIA, 1024)
        self.send(f"SET_BRIGHTNESS ({self.brightness}%)", cmd_brightness(self.brightness))
        return resp

    def get_storage_status(self):
        """GET_STORAGE_STATUS -> returns (total_kb, used_kb, free_kb) or None."""
        resp = self.send("GET_STORAGE_STATUS", CMD_GET_STORAGE, 1024)
        if resp:
            parts = resp.split("-")
            if len(parts) >= 3:
                return int(parts[0]), int(parts[1]), int(parts[2])
        return None

    def get_file_info(self, path):
        """GET_FILE_INFO -> returns file size in bytes, or 0 if not found."""
        resp = self.send(f"GET_FILE_INFO \"{path}\"", build_path_cmd(0x6e, path), 1024)
        if resp and resp.isdigit():
            return int(resp)
        return 0

    def list_dir(self, path="/root/video/"):
        """LIST_DIR -> returns list of filenames."""
        resp = self.send(f"LIST_DIR \"{path}\"", build_path_cmd(0x65, path), 1024)
        # LIST_DIR can leave extra data in the buffer — flush it
        self.flush_input()
        if resp and resp.startswith("result:dir:"):
            content = resp.replace("result:dir:", "").replace("file:", "")
            content = content.rstrip("/")
            if content:
                return content.split("/")
        return []

    def delete_file(self, path):
        """DELETE_FILE."""
        return self.send(f"DELETE_FILE \"{path}\"", build_path_cmd(0x66, path), 1024)

    def restart_device(self, wait=2.0):
        """RESTART_DEVICE (soft restart 0x82) + wait + HELLO."""
        self.send("RESTART_DEVICE", CMD_RESTART)
        print(f"  (waiting {wait}s for restart...)")
        time.sleep(wait)
        resp = self.send("HELLO", CMD_HELLO, 23)
        if not resp:
            # Retry once
            time.sleep(2)
            resp = self.send("HELLO (retry)", CMD_HELLO, 23)
        return resp

    def play_video(self, device_path, loop=True):
        """PLAY_VIDEO -> returns response string."""
        loop_str = "loop" if loop else "once"
        return self.send(
            f"PLAY_VIDEO \"{device_path}\" {loop_str}",
            build_play_video(device_path, loop),
            1024
        )

    def upload_file(self, local_path, device_path):
        """
        Upload a file to device storage.
        CRITICAL: DTR=1 and RTS=1 must be set for upload to work.
        The Windows app sets these control lines before data transfer.
        """
        import os, time
        file_size = os.path.getsize(local_path)

        # Set DTR and RTS (required for upload - discovered from PCAP)
        self.port.dtr = True
        self.port.rts = True
        time.sleep(0.01)

        # CREATE_FILE
        resp = self.send(
            f"CREATE_FILE \"{device_path}\" size={file_size}",
            build_create_file(device_path, file_size),
            1024
        )
        if resp != "create_success":
            print(f"  ERROR: Expected 'create_success', got '{resp}'")
            return False

        # Read file
        with open(local_path, 'rb') as f:
            data = f.read()

        # Pad to match PCAP behavior: total must be multiple of 250
        # AND add extra padding (PCAP shows ~2000 extra bytes beyond file)
        remainder = len(data) % 250
        if remainder != 0:
            data = data + b'\x00' * (250 - remainder)
        # Add 8 extra chunks of 250 zeros (matches gunpla PCAP: 482750 - 480750 = 2000)
        data = data + b'\x00' * 2000

        print(f"  Uploading {file_size} bytes (sending {len(data)} total)...")
        time.sleep(0.05)

        # Write all data
        self.port.write_timeout = None
        self.port.write(data)
        print(f"  Sent {len(data)} bytes")

        # Wait for "file_rev_done" (device writes to flash, can take seconds)
        # The PCAP shows ~22 CLEAR_FEATURE cycles (~5-10 seconds) before response
        print("  Waiting for confirmation (up to 60s)...")
        import time
        self.port.timeout = 1
        for attempt in range(60):
            resp = self.port.read(1024)
            if len(resp) > 0:
                text = resp.rstrip(b'\x00').decode('ascii', errors='replace')
                print(f"  <- raw: {len(resp)} bytes: \"{text}\"")
                if "file_rev_done" in text:
                    self.port.timeout = READ_TIMEOUT
                    return True
                break
            # Small delay between read attempts
            time.sleep(0.5)
        
        self.port.timeout = READ_TIMEOUT
        print(f"  ERROR: No file_rev_done received")
        return False
