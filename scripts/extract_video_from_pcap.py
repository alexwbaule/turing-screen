#!/usr/bin/env python3
"""
Extract video file data from a USB pcapng/pcap capture of Turing Smart Screen 5".

Reads the pcapng file directly (no external libs needed), finds the CREATE_FILE
command (0x6f), then extracts the raw upload bytes that follow.

Works with both Linux (usbmon) and Windows (USBPcap) captures.

Usage: python3 extract_video_from_pcap.py <pcapng_file> [output_file]
"""

import struct
import sys
import os


def read_pcapng_packets(filepath):
    """Read all Enhanced Packet Blocks from a pcapng file."""
    packets = []
    with open(filepath, 'rb') as f:
        data = f.read()

    offset = 0
    while offset < len(data):
        if offset + 8 > len(data):
            break
        block_type = struct.unpack_from('<I', data, offset)[0]
        block_len = struct.unpack_from('<I', data, offset + 4)[0]
        if block_len < 12 or offset + block_len > len(data):
            break

        if block_type == 0x00000006:  # Enhanced Packet Block
            cap_len = struct.unpack_from('<I', data, offset + 20)[0]
            orig_len = struct.unpack_from('<I', data, offset + 24)[0]
            pkt_data = data[offset + 28: offset + 28 + cap_len]
            packets.append((cap_len, orig_len, pkt_data))

        offset += block_len

    return packets


def parse_usb_packet(pkt):
    """
    Parse a USBPcap packet. Returns (endpoint, payload).
    Header format: first 2 bytes LE = header length, byte 21 = endpoint.
    """
    if len(pkt) < 27:
        return None, b''
    hdr_len = struct.unpack_from('<H', pkt, 0)[0]
    if hdr_len < 27 or hdr_len > len(pkt):
        return None, b''
    endpoint = pkt[21]
    payload = pkt[hdr_len:]
    return endpoint, payload


def extract_video(pcapng_path, output_path=None):
    print(f"Reading {pcapng_path}...")
    packets = read_pcapng_packets(pcapng_path)
    print(f"  {len(packets)} packets")

    # Find CREATE_FILE command (0x6f 0xef 0x69)
    create_idx = None
    file_size = None
    file_path = None

    for i, (cap, orig, pkt) in enumerate(packets):
        endpoint, payload = parse_usb_packet(pkt)
        if endpoint is None or endpoint & 0x80:
            continue
        if len(payload) < 14:
            continue
        if payload[0] == 0x6f and payload[1] == 0xef and payload[2] == 0x69:
            path_len = int.from_bytes(payload[3:7], 'big')
            if 10 + path_len + 4 <= len(payload):
                file_path = payload[10:10 + path_len].decode('ascii', errors='replace')
                file_size = int.from_bytes(payload[10 + path_len:10 + path_len + 4], 'little')
                create_idx = i
                print(f"\n  CREATE_FILE at packet {i}")
                print(f"    Path: {file_path}")
                print(f"    Size: {file_size} bytes ({file_size / 1024:.1f} KB, {file_size / 1024 / 1024:.2f} MB)")
                break

    if create_idx is None:
        print("ERROR: No CREATE_FILE command found!")
        sys.exit(1)

    # Collect upload data: skip until "create_success", then collect OUT data
    collecting = False
    video_data = bytearray()
    truncated_frames = 0

    for i in range(create_idx + 1, len(packets)):
        cap, orig, pkt = packets[i]
        endpoint, payload = parse_usb_packet(pkt)
        if endpoint is None:
            continue

        if not collecting:
            # Wait for create_success response
            if (endpoint & 0x80) and len(payload) > 0:
                txt = payload.decode('ascii', errors='replace').rstrip('\x00')
                if 'create_success' in txt:
                    print(f"    <- \"{txt}\"")
                    collecting = True
            continue

        # Skip empty packets (USB status/control)
        if len(payload) == 0:
            continue

        # IN with actual data = end of upload (file_rev_done)
        if endpoint & 0x80:
            txt = payload.decode('ascii', errors='replace').rstrip('\x00')
            if txt:
                print(f"    <- \"{txt}\"")
            break

        # OUT with data = upload chunk
        video_data.extend(payload)
        if cap < orig:
            truncated_frames += 1

    print(f"\n  Collected: {len(video_data)} bytes")
    if truncated_frames > 0:
        print(f"  ⚠ {truncated_frames} frames were truncated by capture (snaplen limit)")
        print(f"    File may be incomplete/corrupted")

    # Trim to exact file size (remove USB padding)
    if len(video_data) >= file_size:
        video_data = video_data[:file_size]
        print(f"  Trimmed to exact file size: {file_size} bytes")
    else:
        missing = file_size - len(video_data)
        print(f"  ⚠ Missing {missing} bytes ({missing * 100 / file_size:.1f}% of file)")

    # Determine output filename
    if output_path is None:
        output_path = os.path.basename(file_path) if file_path else "extracted_video.mp4"

    with open(output_path, 'wb') as f:
        f.write(video_data)
    print(f"  Saved to: {output_path}")

    # Verify MP4 header
    if len(video_data) >= 8:
        atom_type = video_data[4:8].decode('ascii', errors='replace')
        if atom_type == "ftyp":
            print(f"  ✓ Valid MP4 file (ftyp atom detected)")
        else:
            print(f"  ⚠ First atom: \"{atom_type}\" (expected \"ftyp\")")

    return output_path


def analyze_full_sequence(pcapng_path):
    """Analyze and print the full command sequence from a capture."""
    print(f"\n{'='*60}")
    print(f"Full command sequence: {pcapng_path}")
    print(f"{'='*60}\n")

    packets = read_pcapng_packets(pcapng_path)

    cmd_names = {
        0x01: 'HELLO', 0x79: 'STOP_VIDEO', 0x96: 'STOP_MEDIA',
        0x7b: 'SET_BRIGHTNESS', 0x64: 'GET_STORAGE_STATUS',
        0x6e: 'GET_FILE_INFO', 0x65: 'LIST_DIR', 0x66: 'DELETE_FILE',
        0x6f: 'CREATE_FILE', 0x78: 'PLAY_VIDEO', 0x82: 'RESTART_DEVICE',
        0x86: 'PRE_UPDATE_BITMAP', 0xcf: 'QUERY_STATUS',
        0x2c: 'SEPARATOR', 0xc8: 'SEND_BITMAP', 0xca: 'SEND_OVERLAY',
        0xcc: 'UPDATE_BITMAP',
    }

    upload_mode = False
    for i, (cap, orig, pkt) in enumerate(packets):
        endpoint, payload = parse_usb_packet(pkt)
        if endpoint is None or len(payload) == 0:
            continue

        if endpoint & 0x80:  # IN
            txt = payload.decode('ascii', errors='replace').rstrip('\x00')
            if txt:
                print(f"  {i:5d} <- \"{txt[:70]}\"")
                if 'create_success' in txt:
                    upload_mode = True
                elif 'file_rev_done' in txt:
                    upload_mode = False
        else:  # OUT
            if upload_mode:
                # Skip raw upload data display
                continue
            cmd = payload[0]
            name = cmd_names.get(cmd, f'CMD_0x{cmd:02x}')
            extra = ''
            if cmd in (0x6e, 0x65, 0x66) and len(payload) >= 10:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10 + plen <= len(payload):
                    extra = f' "{payload[10:10+plen].decode("ascii", errors="replace")}"'
            elif cmd == 0x6f and len(payload) >= 14:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10 + plen + 4 <= len(payload):
                    path = payload[10:10+plen].decode('ascii', errors='replace')
                    fsize = int.from_bytes(payload[10+plen:10+plen+4], 'little')
                    extra = f' "{path}" size={fsize}'
            elif cmd == 0x78 and len(payload) >= 10:
                plen = int.from_bytes(payload[3:7], 'big')
                loop = payload[7]
                if 10 + plen <= len(payload):
                    path = payload[10:10+plen].decode('ascii', errors='replace')
                    extra = f' loop={loop} "{path}"'
            elif cmd == 0x7b and len(payload) >= 11:
                extra = f' ({payload[10]})'

            print(f"  {i:5d} -> {name}{extra}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 extract_video_from_pcap.py <pcapng_file> [output_file]")
        print("       python3 extract_video_from_pcap.py --analyze <pcapng_file>")
        sys.exit(1)

    if sys.argv[1] == "--analyze":
        if len(sys.argv) < 3:
            print("Usage: python3 extract_video_from_pcap.py --analyze <pcapng_file>")
            sys.exit(1)
        analyze_full_sequence(sys.argv[2])
    else:
        pcapng = sys.argv[1]
        output = sys.argv[2] if len(sys.argv) > 2 else None
        extract_video(pcapng, output)
