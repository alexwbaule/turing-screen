#!/usr/bin/env python3
"""
Analyze upload_full.pcap from scratch.
Goal: understand the EXACT protocol, especially sensor updates.
"""

import struct
import sys
import os

filepath = 'Debug/upload_full.pcapng'
with open(filepath, 'rb') as f:
    raw = f.read()

# Verify: pcap LE magic = 0xa1b2c3d4
magic = struct.unpack_from('<I', raw, 0)[0]
assert magic == 0xa1b2c3d4, f"Not a pcap LE file: magic=0x{magic:08x}"

version_major = struct.unpack_from('<H', raw, 4)[0]
version_minor = struct.unpack_from('<H', raw, 6)[0]
thiszone = struct.unpack_from('<i', raw, 8)[0]
sigfigs = struct.unpack_from('<I', raw, 12)[0]
snaplen = struct.unpack_from('<I', raw, 16)[0]
network = struct.unpack_from('<I', raw, 20)[0]

print(f"pcap v{version_major}.{version_minor}, snaplen={snaplen}, network={network}")
# network 220 = LINKTYPE_USB_LINUX_MMAPPED

# Parse all packets
# Per-packet header: ts_sec(4) ts_usec(4) cap_len(4) orig_len(4) = 16 bytes
# Then USB Linux MMAPPED header: 64 bytes
#   [0:8]   urb_id
#   [8]     event type  (0x53='S'ubmit, 0x43='C'omplete)
#   [9]     xfer type   (0=ISO, 1=INT, 2=CTRL, 3=BULK)
#   [10]    endpoint
#   [11]    devnum
#   [12:14] busnum
#   [14]    flag_setup
#   [15]    flag_data
#   [16:24] ts (8 bytes)
#   [24:28] ts_usec
#   [28:32] status
#   [32:36] length (urb total)
#   [36:40] data_len (captured)
#   [40:48] padding/unused
#   [48:64] setup packet (if CTRL+Submit+flag_setup==0)
#   [64:]   payload data

CMD_NAMES = {
    0x01: 'HELLO', 0x79: 'STOP_VIDEO', 0x96: 'STOP_MEDIA',
    0x7b: 'SET_BRIGHTNESS', 0x7d: 'CMD_0x7D', 0x64: 'GET_STORAGE',
    0x65: 'LIST_DIR', 0x66: 'DELETE_FILE', 0x6e: 'GET_FILE_INFO',
    0x6f: 'CREATE_FILE', 0x78: 'PLAY_VIDEO', 0x82: 'RESTART_DEVICE',
    0x86: 'PRE_UPDATE_BITMAP', 0xc8: 'SEND_BITMAP', 0xca: 'SEND_OVERLAY',
    0xcc: 'UPDATE_BITMAP', 0xcf: 'QUERY_STATUS', 0x83: 'TURNOFF',
    0x84: 'RESTART',
}

offset = 24
packets = []
while offset + 16 <= len(raw):
    ts_sec = struct.unpack_from('<I', raw, offset)[0]
    ts_usec = struct.unpack_from('<I', raw, offset + 4)[0]
    cap_len = struct.unpack_from('<I', raw, offset + 8)[0]
    orig_len = struct.unpack_from('<I', raw, offset + 12)[0]
    pkt = raw[offset + 16: offset + 16 + cap_len]
    packets.append({
        'ts': ts_sec + ts_usec / 1_000_000,
        'cap_len': cap_len,
        'orig_len': orig_len,
        'data': pkt,
    })
    offset += 16 + cap_len

print(f"Total packets: {len(packets)}")

# Extract bulk OUT commands and bulk IN responses
# Build a timeline of what was sent and received
print("\n" + "="*80)
print("FULL COMMAND SEQUENCE (bulk OUT commands + bulk IN responses)")
print("="*80)

sequence = []  # list of (pkt_idx, direction, command_or_response)

for i, pkt in enumerate(packets):
    d = pkt['data']
    if len(d) < 48:
        continue

    event = d[8]
    xfer = d[9]
    ep = d[10]
    data_len = struct.unpack_from('<I', d, 36)[0]

    if xfer != 3:  # only BULK
        continue

    payload = d[64:] if len(d) > 64 else b''

    if event == 0x53 and not (ep & 0x80) and data_len > 0:
        # Bulk OUT submit with data = command or file data
        if data_len == 250 and len(payload) >= 1:
            cmd_byte = payload[0]
            name = CMD_NAMES.get(cmd_byte, f'0x{cmd_byte:02x}')
            extra = ''
            if cmd_byte == 0x2c:
                name = 'SEPARATOR'
            elif cmd_byte == 0x6f and len(payload) >= 14:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10 + plen + 4 <= len(payload):
                    path = payload[10:10+plen].decode('ascii', errors='replace')
                    fsize = int.from_bytes(payload[10+plen:10+plen+4], 'little')
                    extra = f' "{path}" size={fsize}'
            elif cmd_byte == 0x6e and len(payload) >= 10:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10 + plen <= len(payload):
                    extra = f' "{payload[10:10+plen].decode("ascii", errors="replace")}"'
            elif cmd_byte == 0x78 and len(payload) >= 10:
                plen = int.from_bytes(payload[3:7], 'big')
                loop = payload[7]
                if 10 + plen <= len(payload):
                    extra = f' "{payload[10:10+plen].decode("ascii", errors="replace")}" loop={loop}'
            elif cmd_byte == 0x7d and len(payload) >= 12:
                extra = f' raw={payload[:12].hex()}'
            elif cmd_byte == 0x7b and len(payload) >= 11:
                extra = f' value={payload[10]}'
            elif cmd_byte == 0xcc:
                size_field = int.from_bytes(payload[4:7], 'big')
                count = int.from_bytes(payload[10:14], 'big')
                extra = f' size={size_field} count={count}'
            elif cmd_byte == 0xca:
                extra = f' header={payload[:8].hex()}'
            elif cmd_byte == 0xc8:
                extra = f' header={payload[:6].hex()}'
            sequence.append((i, 'OUT', name, extra, payload))
        else:
            # File data (not 250 bytes)
            sequence.append((i, 'OUT', 'FILE_DATA', f' {data_len} bytes', payload))

    elif event == 0x43 and (ep & 0x80) and data_len > 0 and len(payload) > 0:
        # Bulk IN complete = device response
        txt = payload[:80].decode('ascii', errors='replace').rstrip('\x00')
        sequence.append((i, 'IN', txt, '', payload))

# Print the full sequence
last_phase = None
for idx, direction, name, extra, payload in sequence:
    if direction == 'OUT':
        # Determine phase
        if name in ('HELLO',):
            phase = 'INIT'
        elif name in ('GET_STORAGE', 'GET_FILE_INFO', 'LIST_DIR', 'CMD_0x7D'):
            phase = 'CHECK'
        elif name in ('CREATE_FILE', 'FILE_DATA'):
            phase = 'UPLOAD'
        elif name in ('RESTART_DEVICE',):
            phase = 'RESTART'
        elif name in ('PLAY_VIDEO',):
            phase = 'PLAY'
        elif name in ('PRE_UPDATE_BITMAP', 'SEPARATOR', 'SET_BRIGHTNESS', 'SEND_BITMAP', 'SEND_OVERLAY'):
            phase = 'OVERLAY'
        elif name in ('UPDATE_BITMAP', 'QUERY_STATUS'):
            phase = 'SENSORS'
        else:
            phase = 'OTHER'

        if phase != last_phase:
            print(f"\n--- {phase} ---")
            last_phase = phase
        print(f"  [{idx:5d}] >> {name}{extra}")
    else:
        print(f"  [{idx:5d}] << \"{name}\"")


# ===== DEEP ANALYSIS OF SENSOR UPDATES =====
print("\n" + "="*80)
print("SENSOR UPDATE ANALYSIS (0xcc commands after overlay)")
print("="*80)

# Find all UPDATE_BITMAP commands and parse their internal format
update_idx = 0
for i, pkt in enumerate(packets):
    d = pkt['data']
    if len(d) < 64:
        continue
    event = d[8]
    xfer = d[9]
    ep = d[10]
    if xfer != 3 or event != 0x53 or (ep & 0x80):
        continue
    data_len = struct.unpack_from('<I', d, 36)[0]
    payload = d[64:]
    if len(payload) < 14 or payload[0] != 0xcc:
        continue

    size_field = int.from_bytes(payload[4:7], 'big')
    count = int.from_bytes(payload[10:14], 'big')

    # Collect all data for this update (follow packets until QUERY_STATUS or next command)
    img_data = bytearray()
    # First chunk: skip the 250-byte header chunk (14 bytes header + 236 bytes data)
    # Actually: the first 250-byte chunk has 14 bytes of cc header, rest is data
    # But the way the Go code generates it, the first chunk is the full 250-byte cc command
    # and subsequent chunks are 249-byte data chunks
    #
    # From the PCAP perspective: all the 250-byte chunks for one UPDATE_BITMAP
    # are in consecutive USB bulk OUT transfers

    # Collect from this packet + subsequent packets until 0xcf or new command
    img_data.extend(payload[14:data_len])  # data after the 14-byte header in first chunk

    for j in range(i+1, min(i+200, len(packets))):
        p = packets[j]['data']
        if len(p) < 64:
            continue
        ev = p[8]
        xf = p[9]
        ep2 = p[10]
        dl = struct.unpack_from('<I', p, 36)[0]
        pay = p[64:]
        if xf != 3 or ev != 0x53 or (ep2 & 0x80):
            continue
        if dl == 0:
            continue
        if len(pay) > 0 and pay[0] in (0xcf, 0xcc, 0x01, 0x79, 0x96):
            break
        img_data.extend(pay[:dl])

    # Parse the position/width/pixel format
    # Format per line: [position:3BE][width:2BE][pixel_data: width*bpp bytes]
    # End marker: ef 69
    off = 0
    lines = []
    positions = []
    widths = []

    # Try both BGR(3) and BGRA(4)
    # Auto-detect by trying to parse and seeing which makes sense
    for bpp in [3, 4]:
        test_off = 0
        test_lines = 0
        ok = True
        while test_off < min(size_field, len(img_data)) - 2:
            if img_data[test_off] == 0xef and test_off + 1 < len(img_data) and img_data[test_off+1] == 0x69:
                break
            if test_off + 5 > min(size_field, len(img_data)):
                ok = False
                break
            pos = int.from_bytes(img_data[test_off:test_off+3], 'big')
            w = int.from_bytes(img_data[test_off+3:test_off+5], 'big')
            test_off += 5
            if w == 0 or w > 800 or pos > 800*480:
                ok = False
                break
            pixel_bytes = w * bpp
            if test_off + pixel_bytes > min(size_field, len(img_data)):
                ok = False
                break
            test_off += pixel_bytes
            test_lines += 1
        if ok and test_lines >= 2:
            break
    else:
        bpp = 3  # fallback

    # Now parse with detected bpp
    off = 0
    while off < min(size_field, len(img_data)) - 2:
        if img_data[off] == 0xef and off + 1 < len(img_data) and img_data[off+1] == 0x69:
            break
        if off + 5 > min(size_field, len(img_data)):
            break
        pos = int.from_bytes(img_data[off:off+3], 'big')
        w = int.from_bytes(img_data[off+3:off+5], 'big')
        off += 5
        if w == 0 or w > 800:
            break
        pixel_bytes = w * bpp
        if off + pixel_bytes > min(size_field, len(img_data)):
            break

        # Decode position
        row_800 = pos // 800
        col_800 = pos % 800

        # Also try 803 stride
        row_803 = pos // 803
        col_803 = pos % 803

        lines.append({
            'position': pos,
            'width': w,
            'row_800': row_800,
            'col_800': col_800,
            'row_803': row_803,
            'col_803': col_803,
        })
        positions.append(pos)
        widths.append(w)
        off += pixel_bytes

    if len(lines) < 2:
        continue

    # Analyze stride: check differences between consecutive positions
    pos_diffs = []
    for k in range(1, len(lines)):
        pos_diffs.append(lines[k]['position'] - lines[k-1]['position'])

    # Calculate what the stride should be
    # If position = row * STRIDE + col, then consecutive rows differ by STRIDE
    # Filter out same-row positions (diff = pixel advance within same row)
    row_diffs = [d for d in pos_diffs if d > 100]  # significant jumps = row changes

    avg_stride = 0
    if row_diffs:
        avg_stride = sum(row_diffs) // len(row_diffs)

    print(f"\nUpdate #{update_idx} (pkt {i}): size_field={size_field} count={count} bpp={bpp} lines={len(lines)}")
    print(f"  First 5 lines:")
    for k, line in enumerate(lines[:5]):
        print(f"    line {k}: pos={line['position']:6d} w={line['width']:3d}  "
              f"800→ row={line['row_800']:3d} col={line['col_800']:3d}  "
              f"803→ row={line['row_803']:3d} col={line['col_803']:3d}")

    print(f"  Position diffs (first 10): {pos_diffs[:10]}")
    print(f"  Row-changing diffs (>100): {row_diffs[:10]}")
    if row_diffs:
        print(f"  Average row stride: {avg_stride}")
        print(f"  Stride consistent with 800? {all(d == 800 for d in row_diffs)}")
        print(f"  Stride consistent with 803? {all(d == 803 for d in row_diffs)}")

    # Check the Go code's formula: position = (x0+h)*display.Width + y0
    # In LANDSCAPE mode: x0=y, y0=x (line 88: x0, y0 = y, x)
    # So position = (sensor_y + row_offset) * display.Width + sensor_x
    # Stride between rows = display.Width

    # Check col consistency for both strides
    cols_800 = [l['col_800'] for l in lines]
    cols_803 = [l['col_803'] for l in lines]
    print(f"  Cols with stride 800: all same? {len(set(cols_800))==1} values={set(cols_800)}")
    print(f"  Cols with stride 803: all same? {len(set(cols_803))==1} values={set(cols_803)}")

    # Rows check
    rows_800 = [l['row_800'] for l in lines]
    rows_803 = [l['row_803'] for l in lines]
    row_diffs_800 = [rows_800[k+1]-rows_800[k] for k in range(len(rows_800)-1)]
    row_diffs_803 = [rows_803[k+1]-rows_803[k] for k in range(len(rows_803)-1)]
    print(f"  Row diffs with stride 800: {row_diffs_800[:10]} (consecutive? {all(d==1 for d in row_diffs_800 if d>0)})")
    print(f"  Row diffs with stride 803: {row_diffs_803[:10]} (consecutive? {all(d==1 for d in row_diffs_803 if d>0)})")

    update_idx += 1
    if update_idx >= 15:
        break

print(f"\nAnalyzed {update_idx} sensor updates")
