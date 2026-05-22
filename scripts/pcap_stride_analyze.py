#!/usr/bin/env python3
"""
Parse sensor UPDATE_BITMAP data from the PCAP correctly.
The first 250 bytes are the cc header chunk (padded).
Actual image data (position/width/pixel) starts at byte 250.
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

# Sensor update packets (found from previous analysis)
# pkt 8668: 0xcc size=17507 (first gunpla session sensor updates)
# pkt 8758: 0xcc size=18614
# pkt 8862: 0xcc size=16644

sensor_pkts = [8668, 8758, 8862]

for pkt_idx in sensor_pkts:
    d = packets[pkt_idx]
    data_len = struct.unpack_from('<I', d, 36)[0]
    payload = d[64:]

    # Parse 0xcc header
    size_field = int.from_bytes(payload[4:7], 'big')
    count = int.from_bytes(payload[10:14], 'big')

    # Collect full data: this packet + subsequent non-command packets
    img_raw = bytearray(payload[:data_len])

    for j in range(pkt_idx + 1, min(pkt_idx + 200, len(packets))):
        p = packets[j]
        if len(p) < 64:
            continue
        ev = p[8]
        xf = p[9]
        ep = p[10]
        dl = struct.unpack_from('<I', p, 36)[0]
        pay = p[64:]
        if xf != 3 or ev != 0x53 or (ep & 0x80):
            continue
        if dl == 0:
            continue
        if len(pay) > 0 and pay[0] in (0xcf, 0xcc, 0x01, 0x79, 0x96, 0x87):
            break
        img_raw.extend(pay[:dl])

    print(f"\n{'='*70}")
    print(f"pkt {pkt_idx}: UPDATE_BITMAP size_field={size_field} count={count}")
    print(f"Total raw bytes: {len(img_raw)}")

    # The data is structured as 250-byte serial frames:
    # - Frame 0: [cc header 14B][zeros 236B] = 250 bytes
    # - Frame 1-N: [payload 249B][zero 1B] = 250 bytes each
    #
    # To get pure image data, we need to de-frame:
    # Skip first 250 bytes (header frame), then take 249 bytes from each 250-byte frame

    # Method 1: Simple - just skip the first 250 bytes and use the rest
    # This works because the padding bytes are zeros, and the position/width
    # parser handles zeros gracefully at boundaries
    img_data_after_header = img_raw[250:]  # skip header chunk

    # Method 2: Proper de-framing - take first 249 bytes of each 250-byte frame
    img_data_deframed = bytearray()
    for i in range(1, len(img_raw) // 250 + 1):
        start = i * 250
        end = start + 249
        if end > len(img_raw):
            end = len(img_raw)
        if start < len(img_raw):
            img_data_deframed.extend(img_raw[start:end])

    # Method 3: Even simpler - look at what the Go code generates
    # GeneratePartialImage output is directly what goes into m.payload
    # Then GetBytes frames it into 250-byte chunks
    # So the pure payload = de-frame all chunks
    # But the Windows app might NOT use serial framing!
    # It might send the raw image data directly

    # Let's try ALL methods and see which one parses correctly
    for method_name, img_data in [
        ("raw[250:]", img_data_after_header),
        ("deframed", img_data_deframed),
        ("raw[14:]", img_raw[14:]),  # skip only cc header
    ]:
        print(f"\n  Method: {method_name} ({len(img_data)} bytes)")

        for bpp in [3, 4]:
            off = 0
            lines = []
            while off < min(size_field, len(img_data)) - 2:
                if off + 2 < len(img_data) and img_data[off] == 0xef and img_data[off+1] == 0x69:
                    break
                if off + 5 > min(size_field, len(img_data)):
                    break
                pos = int.from_bytes(img_data[off:off+3], 'big')
                w = int.from_bytes(img_data[off+3:off+5], 'big')
                off += 5
                if w == 0 or w > 800 or pos > 800*480:
                    break
                pix = w * bpp
                if off + pix > min(size_field, len(img_data)):
                    break
                lines.append({'pos': pos, 'w': w})
                off += pix

            if len(lines) >= 2:
                pos_diffs = [lines[k+1]['pos'] - lines[k]['pos'] for k in range(len(lines)-1)]
                row_diffs = [d for d in pos_diffs if d > 50]

                # Check strides
                stride_800 = all(l['pos'] % 800 == lines[0]['pos'] % 800 for l in lines)
                stride_803 = all(l['pos'] % 803 == lines[0]['pos'] % 803 for l in lines)

                cols_800 = [l['pos'] % 800 for l in lines]
                rows_800 = [l['pos'] // 800 for l in lines]
                row_diffs_800 = [rows_800[k+1]-rows_800[k] for k in range(len(rows_800)-1)]
                consecutive_800 = all(d == 1 for d in row_diffs_800 if d > 0)

                cols_803 = [l['pos'] % 803 for l in lines]
                rows_803 = [l['pos'] // 803 for l in lines]
                row_diffs_803 = [rows_803[k+1]-rows_803[k] for k in range(len(rows_803)-1)]
                consecutive_803 = all(d == 1 for d in row_diffs_803 if d > 0)

                print(f"    bpp={bpp}: {len(lines)} lines parsed!")
                print(f"    pos_diffs: {pos_diffs[:10]}")
                if row_diffs:
                    print(f"    row_diffs: {row_diffs[:8]} avg={sum(row_diffs)//len(row_diffs)}")
                print(f"    stride 800: cols_same={stride_800} cols={set(cols_800[:5])} rows_consecutive={consecutive_800}")
                print(f"    stride 803: cols_same={stride_803} cols={set(cols_803[:5])} rows_consecutive={consecutive_803}")
                print(f"    First 3: ", end="")
                for k in range(min(3, len(lines))):
                    p = lines[k]['pos']
                    print(f"pos={p} (800:r{p//800}c{p%800} 803:r{p//803}c{p%803}) w={lines[k]['w']}  ", end="")
                print()
                break
        else:
            print(f"    Failed to parse with both bpp=3 and bpp=4")
