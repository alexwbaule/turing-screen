#!/usr/bin/env python3
"""
Deep analysis of sensor update data in the PCAP.
The Windows app sends raw USB data, not serial-framed commands.
We need to find and decode the sensor update format.
"""

import struct

filepath = 'Debug/upload_full.pcapng'
with open(filepath, 'rb') as f:
    raw = f.read()

offset = 24
packets = []
while offset + 16 <= len(raw):
    cap_len = struct.unpack_from('<I', raw, offset + 8)[0]
    pkt = raw[offset + 16: offset + 16 + cap_len]
    packets.append(pkt)
    offset += 16 + cap_len

print(f"Total packets: {len(packets)}")

# Focus on the LAST gunpla upload (most complete session)
# From previous analysis:
#   pkt 8348: PLAY_VIDEO gunpla.mp4 -> play_video_success
#   pkt 8360: PRE_UPDATE_BITMAP
#   pkt 8373: SEPARATOR
#   pkt 8376: SET_BRIGHTNESS
#   Then overlay data + sensor updates

# Let me look at EVERY bulk OUT packet from 8376 onwards and dump the first bytes

print("\n=== RAW BULK OUT PACKETS AFTER SET_BRIGHTNESS (pkt 8376+) ===\n")

for i in range(8376, min(8950, len(packets))):
    pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer = pkt[9]
    ep = pkt[10]
    data_len = struct.unpack_from('<I', pkt, 36)[0]

    # Only bulk OUT submits with data
    if xfer != 3 or event != 0x53 or (ep & 0x80) or data_len == 0:
        continue

    payload = pkt[64:]
    if len(payload) == 0:
        continue

    first_byte = payload[0]

    # Classify
    if data_len == 250:
        cmd_names = {
            0x01: 'HELLO', 0x79: 'STOP_VIDEO', 0x96: 'STOP_MEDIA',
            0x7b: 'SET_BRIGHTNESS', 0x86: 'PRE_UPDATE', 0x2c: 'SEPARATOR',
            0xca: 'SEND_OVERLAY', 0xc8: 'SEND_BITMAP', 0xcc: 'UPDATE_BITMAP',
            0xcf: 'QUERY_STATUS', 0x82: 'RESTART', 0x7d: 'CMD_0x7D',
        }
        name = cmd_names.get(first_byte, f'CMD_0x{first_byte:02x}')
        extra = ''
        if first_byte == 0xca:
            extra = f' header={payload[:8].hex()}'
        elif first_byte == 0xcc:
            size_field = int.from_bytes(payload[4:7], 'big')
            count = int.from_bytes(payload[10:14], 'big')
            extra = f' size={size_field} count={count}'
        print(f"  [{i:5d}] 250B  {name}{extra}")
    else:
        # Non-250 byte data — this is raw data
        # Show first 20 bytes as hex
        preview = payload[:min(20, data_len)].hex()
        print(f"  [{i:5d}] {data_len:6d}B raw: {preview}{'...' if data_len > 20 else ''}")


# Now find the overlay boundary and extract sensor update data
print("\n\n=== FINDING OVERLAY + SENSOR UPDATES ===\n")

# Strategy: find the SECOND "seq_png_init_sucess" (the PCAP shows 2 overlay sends)
# After that, everything is sensor updates + QUERY_STATUS

# Find all bulk IN responses
responses = []
for i in range(8300, 9000):
    pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer = pkt[9]
    ep = pkt[10]
    data_len = struct.unpack_from('<I', pkt, 36)[0]
    if xfer != 3 or event != 0x43 or not (ep & 0x80) or data_len == 0:
        continue
    payload = pkt[64:]
    txt = payload[:80].decode('ascii', errors='replace').rstrip('\x00')
    if txt:
        responses.append((i, txt))

print("Bulk IN responses after pkt 8300:")
for idx, txt in responses:
    print(f"  [{idx:5d}] \"{txt}\"")


# Now let me look at the raw data format of the sensor update packets
# The sensor updates come after the last "seq_png_init_sucess"
# Looking at the responses, the pattern is:
#   seq_png_init_sucess (end of overlay 1)
#   ... data ...
#   seq_png_init_sucess (end of overlay 2)
#   needReSend:0|renderCnt:N (after QUERY_STATUS)
#   ... sensor update data ...
#   needReSend:0|renderCnt:N
#   ... sensor update data ...

# Find the last seq_png_init_sucess
last_overlay_end = None
for idx, txt in responses:
    if 'seq_png' in txt:
        last_overlay_end = idx

if last_overlay_end:
    print(f"\nLast overlay response at pkt {last_overlay_end}")
    print(f"\nSensor update packets after pkt {last_overlay_end}:")

    # The sensor updates are raw data packets that don't start with recognized commands
    # They might be UPDATE_BITMAP (0xcc) but not padded to 250 bytes
    # Or they might be the raw image data without command framing

    # Let me look at the structure: after last overlay, we see:
    # - QUERY_STATUS (250B, 0xcf) -> needReSend response
    # - FILE_DATA (sensor update image data)
    # - QUERY_STATUS -> needReSend response
    # - FILE_DATA ...

    # The sensor update data between two QUERY_STATUS is one update

    # Collect sensor update data blocks
    updates = []
    current_update = None

    for i in range(last_overlay_end + 1, min(9000, len(packets))):
        pkt = packets[i]
        if len(pkt) < 64:
            continue
        event = pkt[8]
        xfer = pkt[9]
        ep = pkt[10]
        data_len = struct.unpack_from('<I', pkt, 36)[0]
        if data_len == 0:
            continue
        payload = pkt[64:]

        # Bulk OUT
        if xfer == 3 and event == 0x53 and not (ep & 0x80):
            if data_len == 250 and payload[0] == 0xcf:
                # QUERY_STATUS — end of current update
                if current_update:
                    updates.append(current_update)
                    current_update = None
            else:
                # Data packet (could be 250B command or raw data)
                if current_update is None:
                    current_update = {'start_pkt': i, 'data': bytearray()}
                current_update['data'].extend(payload[:data_len])

        # Bulk IN response
        if xfer == 3 and event == 0x43 and (ep & 0x80):
            txt = payload[:80].decode('ascii', errors='replace').rstrip('\x00')
            if 'needReSend' in txt:
                if current_update:
                    updates.append(current_update)
                    current_update = None

    print(f"\nFound {len(updates)} sensor updates")

    for ui, update in enumerate(updates):
        data = update['data']
        start_pkt = update['start_pkt']
        print(f"\n--- Update #{ui} (pkt {start_pkt}, {len(data)} bytes) ---")
        print(f"  First 40 bytes: {data[:40].hex()}")

        # Try to parse as the Go code's UPDATE_BITMAP format:
        # The Go code generates: [cc ef 69 00][size:3BE][pad:3][count:4BE] + 250B chunks of image data
        # But through serial, each chunk is padded to 250 bytes
        # Through USB raw, it might be different

        # Check if this starts with 0xcc (UPDATE_BITMAP header)
        if len(data) >= 14 and data[0] == 0xcc:
            size_field = int.from_bytes(data[4:7], 'big')
            count = int.from_bytes(data[10:14], 'big')
            print(f"  Starts with 0xcc: size_field={size_field} count={count}")
            print(f"  Header: {data[:14].hex()}")

            # The actual image data starts after the 250-byte header chunk
            # In the Go code: first chunk is 250B header, then 249B data chunks
            # But in USB: it might be all contiguous

            # Skip to where the image data should be
            # The header is the first 250 bytes (padded), then data
            # But actually in the Go format, position/width data starts immediately
            # after the 14-byte header within the first 250B chunk

            # Let me try parsing from byte 14 (after header) as the line format:
            # [position:3BE][width:2BE][pixel_data]
            img_data = data[14:]  # skip cc header
            print(f"  Image data (after 14B header): {len(img_data)} bytes")

        elif len(data) >= 8 and data[0] == 0xca:
            print(f"  Starts with 0xca (SEND_OVERLAY): {data[:8].hex()}")
            continue

        elif len(data) >= 8 and data[0] == 0xc8:
            print(f"  Starts with 0xc8 (SEND_BITMAP)")
            continue

        else:
            print(f"  First byte: 0x{data[0]:02x} (not cc/ca/c8)")
            # Maybe the sensor update data is raw position+pixel data without cc header
            # Let's try to parse as position/width format directly
            img_data = data

        # Try to parse as position/width/pixel line format
        # If this doesn't start with 0xcc, it might be the raw line data
        if 'img_data' not in dir():
            img_data = data

        off = 0
        lines = []
        bpp_detected = 0

        for bpp in [4, 3]:
            test_off = 0
            test_ok = True
            test_lines = 0
            while test_off < min(len(img_data), 10000) - 2:
                if img_data[test_off] == 0xef and test_off + 1 < len(img_data) and img_data[test_off+1] == 0x69:
                    break
                if test_off + 5 > len(img_data):
                    test_ok = False
                    break
                pos = int.from_bytes(img_data[test_off:test_off+3], 'big')
                w = int.from_bytes(img_data[test_off+3:test_off+5], 'big')
                test_off += 5
                if w == 0 or w > 800 or pos > 400000:
                    test_ok = False
                    break
                pix = w * bpp
                if test_off + pix > len(img_data):
                    test_ok = False
                    break
                test_off += pix
                test_lines += 1
            if test_ok and test_lines >= 2:
                bpp_detected = bpp
                break

        if bpp_detected == 0:
            print(f"  Could not parse as position/width/pixel format")
            continue

        # Parse with detected bpp
        off = 0
        while off < min(len(img_data), 100000) - 2:
            if img_data[off] == 0xef and off + 1 < len(img_data) and img_data[off+1] == 0x69:
                break
            if off + 5 > len(img_data):
                break
            pos = int.from_bytes(img_data[off:off+3], 'big')
            w = int.from_bytes(img_data[off+3:off+5], 'big')
            off += 5
            if w == 0 or w > 800:
                break
            pix = w * bpp_detected
            if off + pix > len(img_data):
                break

            row_800 = pos // 800
            col_800 = pos % 800
            row_803 = pos // 803
            col_803 = pos % 803

            lines.append({'pos': pos, 'w': w, 'row800': row_800, 'col800': col_800,
                          'row803': row_803, 'col803': col_803})
            off += pix

        if len(lines) < 2:
            print(f"  Parsed {len(lines)} lines, not enough")
            continue

        # Analyze stride
        pos_diffs = [lines[k+1]['pos'] - lines[k]['pos'] for k in range(len(lines)-1)]
        row_diffs = [d for d in pos_diffs if d > 50]

        cols_800 = set(l['col800'] for l in lines)
        cols_803 = set(l['col803'] for l in lines)
        rows_800 = [l['row800'] for l in lines]
        rows_803 = [l['row803'] for l in lines]

        print(f"  Parsed {len(lines)} lines, bpp={bpp_detected}")
        print(f"  First 3 lines:")
        for k in range(min(3, len(lines))):
            l = lines[k]
            print(f"    pos={l['pos']:6d} w={l['w']:3d} | "
                  f"800: row={l['row800']:3d} col={l['col800']:3d} | "
                  f"803: row={l['row803']:3d} col={l['col803']:3d}")

        print(f"  Position diffs: {pos_diffs[:8]}")
        if row_diffs:
            print(f"  Row-changing diffs: {row_diffs[:8]}")
            print(f"  Average stride: {sum(row_diffs)//len(row_diffs)}")
        print(f"  Cols(800): {cols_800}  all_same={len(cols_800)==1}")
        print(f"  Cols(803): {cols_803}  all_same={len(cols_803)==1}")
        print(f"  Row diffs(800): {[rows_800[k+1]-rows_800[k] for k in range(min(5,len(rows_800)-1))]}")
        print(f"  Row diffs(803): {[rows_803[k+1]-rows_803[k] for k in range(min(5,len(rows_803)-1))]}")
