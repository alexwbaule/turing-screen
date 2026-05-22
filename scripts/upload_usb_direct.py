#!/usr/bin/env python3
"""
Upload video via direct USB (libusb), bypassing CDC ACM serial driver.

The Windows app uses LibUsbDotNet to write directly to USB bulk endpoints.
This script does the same on Linux using pyusb.

Usage: sudo python3 upload_usb_direct.py <local_file.mp4> [device_filename]
"""

import usb.core
import usb.util
import struct
import sys
import os
import time
import math

VENDOR_ID = 0x1d6b
PRODUCT_ID = 0x0106
EP_OUT = 0x01  # Bulk OUT endpoint
EP_IN = 0x81   # Bulk IN endpoint
TIMEOUT = 5000  # ms


def pad(cmd, size=250):
    return cmd + b'\x00' * (size - len(cmd))


def send_cmd(dev, name, cmd_bytes, response_size=0):
    """Send a 250-byte padded command and optionally read response."""
    # Flush any pending data first
    try:
        while True:
            dev.read(EP_IN, 1024, timeout=50)
    except usb.core.USBTimeoutError:
        pass

    dev.write(EP_OUT, pad(cmd_bytes), timeout=TIMEOUT)

    if response_size > 0:
        time.sleep(0.05)
        resp = dev.read(EP_IN, response_size, timeout=TIMEOUT)
        text = bytes(resp).rstrip(b'\x00').decode('ascii', errors='replace')
        print(f"  {name} -> \"{text}\"")
        return text
    else:
        print(f"  {name}")
        time.sleep(0.05)
        return None


def build_path_cmd(cmd_id, path):
    path_bytes = path.encode('ascii')
    payload = bytes([cmd_id, 0xef, 0x69])
    payload += len(path_bytes).to_bytes(4, 'big')
    payload += b'\x00\x00\x00'
    payload += path_bytes
    return payload


def build_create_file(path, file_size):
    path_bytes = path.encode('ascii')
    payload = bytes([0x6f, 0xef, 0x69])
    payload += len(path_bytes).to_bytes(4, 'big')
    payload += b'\x00\x00\x00'
    payload += path_bytes
    payload += file_size.to_bytes(4, 'little')
    return payload


def main():
    if len(sys.argv) < 2:
        print("Usage: sudo python3 upload_usb_direct.py <local_file.mp4> [device_filename]")
        sys.exit(1)

    local_path = sys.argv[1]
    if not os.path.exists(local_path):
        print(f"ERROR: File not found: {local_path}")
        sys.exit(1)

    file_size = os.path.getsize(local_path)
    device_filename = sys.argv[2] if len(sys.argv) > 2 else os.path.basename(local_path)
    device_path = f"/root/video/{device_filename}"

    print(f"=== USB Direct Upload ===")
    print(f"  Local:  {local_path} ({file_size} bytes, {file_size/1024:.1f} KB)")
    print(f"  Device: {device_path}")
    print()

    # Find device
    dev = usb.core.find(idVendor=VENDOR_ID, idProduct=PRODUCT_ID)
    if dev is None:
        print("ERROR: Device not found! Is it connected?")
        print("  (Make sure CDC ACM driver is not claiming it, or run as root)")
        sys.exit(1)

    print(f"  Found: {dev.manufacturer} {dev.product} (serial: {dev.serial_number})")

    # Detach kernel driver if attached (CDC ACM)
    for cfg in dev:
        for intf in cfg:
            if dev.is_kernel_driver_active(intf.bInterfaceNumber):
                print(f"  Detaching kernel driver from interface {intf.bInterfaceNumber}")
                dev.detach_kernel_driver(intf.bInterfaceNumber)

    # Reset the device to clear any CDC ACM state
    print("  Resetting USB device...")
    dev.reset()
    time.sleep(1)

    # Re-find after reset
    dev = usb.core.find(idVendor=VENDOR_ID, idProduct=PRODUCT_ID)
    if dev is None:
        print("ERROR: Device not found after reset!")
        sys.exit(1)

    # Detach again after reset
    for cfg in dev:
        for intf in cfg:
            if dev.is_kernel_driver_active(intf.bInterfaceNumber):
                dev.detach_kernel_driver(intf.bInterfaceNumber)

    # Set configuration and claim the data interface (interface 1 = CDC Data)
    dev.set_configuration()
    usb.util.claim_interface(dev, 0)  # CDC Control interface
    usb.util.claim_interface(dev, 1)  # CDC Data interface

    # Replicate Windows app initialization sequence:
    # SET_CONTROL_LINE_STATE DTR=0 RTS=0
    dev.ctrl_transfer(0x21, 0x22, 0x0000, 0x0000, None)
    # SET_LINE_CODING: 115200, 8N1
    line_coding = struct.pack('<IBBB', 115200, 0, 0, 8)
    dev.ctrl_transfer(0x21, 0x20, 0x0000, 0x0000, line_coding)
    # SET_CONTROL_LINE_STATE DTR=1 RTS=1
    dev.ctrl_transfer(0x21, 0x22, 0x0003, 0x0000, None)
    # SET_LINE_CODING again
    dev.ctrl_transfer(0x21, 0x20, 0x0000, 0x0000, line_coding)
    # SET_CONTROL_LINE_STATE DTR=1 RTS=1 again
    dev.ctrl_transfer(0x21, 0x22, 0x0003, 0x0000, None)

    print(f"  USB configured, DTR=1 RTS=1, 115200 8N1")
    print()

    try:
        # HELLO (first one may timeout after detach)
        hello = bytes([0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3])
        resp = send_cmd(dev, "HELLO", hello, 1024)
        if not resp:
            print("ERROR: No HELLO response")
            sys.exit(1)

        # STOP_VIDEO
        send_cmd(dev, "STOP_VIDEO", bytes([0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]))

        # STOP_MEDIA
        send_cmd(dev, "STOP_MEDIA", bytes([0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]), 1024)

        # SET_BRIGHTNESS
        bright = int((20 / 100.0) * 255)
        send_cmd(dev, f"SET_BRIGHTNESS ({bright})", bytes([0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, bright]))

        # GET_STORAGE_STATUS
        resp = send_cmd(dev, "GET_STORAGE_STATUS", bytes([0x64, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]), 1024)

        # CMD_0x7d - sent before every upload in the Windows app
        # Possibly SET_TRANSFER_MODE or similar. Always: 7d ef 69 00 00 00 05 00 00 00 aa 00
        send_cmd(dev, "CMD_0x7D (pre-upload)", bytes([0x7d, 0xef, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0xaa, 0x00]))

        # GET_FILE_INFO
        resp = send_cmd(dev, f"GET_FILE_INFO \"{device_path}\"", build_path_cmd(0x6e, device_path), 1024)
        if resp and resp != "0":
            print(f"    File exists ({resp} bytes), deleting...")
            send_cmd(dev, "DELETE_FILE", build_path_cmd(0x66, device_path), 1024)
            send_cmd(dev, "GET_STORAGE_STATUS", bytes([0x64, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]), 1024)

        # STOP_VIDEO + STOP_MEDIA (as seen in PCAP before upload)
        send_cmd(dev, "STOP_VIDEO", bytes([0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]))
        send_cmd(dev, "STOP_MEDIA", bytes([0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]), 1024)

        # CREATE_FILE
        resp = send_cmd(dev, f"CREATE_FILE size={file_size}", build_create_file(device_path, file_size), 1024)
        if resp != "create_success":
            print(f"  ERROR: Expected 'create_success', got '{resp}'")
            sys.exit(1)

        # Upload file data
        with open(local_path, 'rb') as f:
            data = f.read()

        print(f"\n  Uploading {file_size} bytes...")

        # Wait 50ms for device to prepare (as seen in PCAP)
        time.sleep(0.05)

        # Write data in 512-byte chunks while continuously polling EP 0x84
        # The PCAP shows CLEAR_FEATURE on EP 0x84 happening throughout the
        # entire session (it's the CDC ACM interrupt endpoint polling).
        # This may be required for the device to function correctly.
        import threading

        # Start continuous CLEAR_FEATURE polling in background
        stop_polling = threading.Event()
        def poll_interrupt():
            while not stop_polling.is_set():
                try:
                    dev.ctrl_transfer(0x02, 0x01, 0x0000, 0x0084, None)
                except:
                    pass
                time.sleep(0.01)  # ~100 Hz, matching PCAP pattern

        poller = threading.Thread(target=poll_interrupt, daemon=True)
        poller.start()

        CHUNK = 512
        sent = 0
        t0 = time.time()
        while sent < len(data):
            end = min(sent + CHUNK, len(data))
            dev.write(EP_OUT, data[sent:end], timeout=30000)
            sent = end
            pct = (sent * 100) // len(data)
            if sent % 50000 < CHUNK:
                elapsed = time.time() - t0
                speed = sent / elapsed / 1024 / 1024 if elapsed > 0 else 0
                print(f"    {pct}% ({sent}/{len(data)}) {speed:.1f} MB/s")

        elapsed = time.time() - t0
        print(f"  Sent {sent} bytes in {elapsed:.2f}s ({sent/elapsed/1024/1024:.1f} MB/s)")

        # Wait for file_rev_done while continuing to poll
        print("  Waiting for file_rev_done (up to 60s)...")
        try:
            resp = dev.read(EP_IN, 1024, timeout=60000)
            text = bytes(resp).rstrip(b'\x00').decode('ascii', errors='replace')
            print(f"  <- \"{text}\"")
        except usb.core.USBTimeoutError:
            print("  <- timeout (60s)")
        except usb.core.USBError as e:
            print(f"  <- USB error: {e}")
        finally:
            stop_polling.set()
            poller.join(timeout=2)
        # Verify: GET_FILE_INFO + GET_STORAGE + LIST_DIR
        print()
        resp = send_cmd(dev, f"GET_FILE_INFO \"{device_path}\"", build_path_cmd(0x6e, device_path), 1024)
        print(f"    Uploaded size: {resp}")

        resp = send_cmd(dev, "GET_STORAGE_STATUS", bytes([0x64, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01]), 1024)
        if resp:
            parts = resp.split("-")
            if len(parts) >= 3:
                print(f"    Storage: {parts[0]} KB total, {parts[1]} KB used, {parts[2]} KB free")

        # LIST_DIR
        send_cmd(dev, "LIST_DIR", build_path_cmd(0x65, "/root/video/"), 1024)
        # Flush extra list data
        try:
            while True:
                dev.read(EP_IN, 1024, timeout=200)
        except:
            pass

        # Verify
        resp = send_cmd(dev, f"GET_FILE_INFO \"{device_path}\"", build_path_cmd(0x6e, device_path), 1024)
        print(f"\n=== UPLOAD {'SUCCESS' if resp == str(file_size) else 'VERIFY FAILED'} ===")
        print(f"  Device: {resp} bytes, Expected: {file_size} bytes")

    finally:
        # Re-attach kernel driver
        usb.util.dispose_resources(dev)
        print("\n  Done. You may need to replug the device for serial access.")


if __name__ == "__main__":
    main()
