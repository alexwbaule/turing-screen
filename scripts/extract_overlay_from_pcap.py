#!/usr/bin/env python3
"""
Extract overlay and sensor update images from upload_full.pcapng.
Saves them as PNG files to see what the Windows app sends.
"""

import struct
from PIL import Image

filepath = 'Debug/upload_full.pcapng'
with open(filepath, 'rb') as f:
    data = f.read()

# Parse legacy PCAP
offset = 24
packets = []
while offset < len(data):
    if offset + 16 > len(data):
        break
    cap_len = struct.unpack_from('<I', data, offset + 8)[0]
    pkt_data = data[offset + 16: offset + 16 + cap_len]
    packets.append(pkt_data)
    offset += 16 + cap_len

print(f"Total packets: {len(packets)}")

# Find the SEND_OVERLAY (0xca) after PLAY_VIDEO for gunpla
# PLAY_VIDEO is at pkt 8348, overlay starts around 8382

# Collect the overlay data (0xca command + bitmap data)
collecting = False
overlay_data = bytearray()

for i in range(8360, 8510):
    pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer_type = pkt[9]
    endpoint = pkt[10]
    data_len = struct.unpack_from('<I', pkt, 36)[0]
    payload = pkt[64:]

    if xfer_type != 3 or event != 0x53 or (endpoint & 0x80):
        continue
    if data_len == 0 or len(payload) == 0:
        continue

    # Look for the 0xca header (SEND_OVERLAY)
    if not collecting and payload[0] == 0xca:
        collecting = True
        # Skip the header (first 250-byte chunk is the 0xca command)
        # The actual bitmap data starts in subsequent chunks
        continue

    if not collecting and payload[0] == 0x2c:
        # Separator - skip
        continue

    if not collecting and payload[0] == 0x86:
        # PRE_UPDATE_BITMAP - skip
        continue

    if not collecting and payload[0] == 0x7b:
        # SET_BRIGHTNESS - skip
        continue

    if collecting:
        # Check if we hit seq_png_init_sucess response or next command
        if payload[0] == 0xcf:  # QUERY_STATUS = end of overlay
            break
        if payload[0] == 0xca:  # Second overlay send
            break
        overlay_data.extend(payload[:data_len])

print(f"\nOverlay data collected: {len(overlay_data)} bytes")
print(f"Expected for 800x480 BGRA: {800*480*4} bytes")
print(f"Expected for 800x480 BGR: {800*480*3} bytes")

# Try to interpret as BGRA (4 bytes per pixel)
if len(overlay_data) >= 800*480*4:
    img = Image.frombytes('RGBA', (800, 480), bytes(overlay_data[:800*480*4]), 'raw', 'BGRA')
    img.save('Debug/overlay_bgra.png')
    print(f"Saved Debug/overlay_bgra.png (BGRA)")

# Try BGR (3 bytes per pixel)  
if len(overlay_data) >= 800*480*3:
    img = Image.frombytes('RGB', (800, 480), bytes(overlay_data[:800*480*3]), 'raw', 'BGR')
    img.save('Debug/overlay_bgr.png')
    print(f"Saved Debug/overlay_bgr.png (BGR)")

# Now extract a sensor UPDATE_BITMAP
# These start after the overlay, around pkt 8668
print("\n\n=== Extracting sensor UPDATE_BITMAP ===")

for i in range(8660, 8710):
    pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer_type = pkt[9]
    endpoint = pkt[10]
    data_len = struct.unpack_from('<I', pkt, 36)[0]
    payload = pkt[64:]

    if xfer_type != 3 or event != 0x53 or (endpoint & 0x80):
        continue
    if len(payload) < 14:
        continue

    # UPDATE_BITMAP starts with 0xcc
    if payload[0] == 0xcc:
        # Header: cc ef 69 00 [size:3] [pad:3] [count:4]
        size_field = int.from_bytes(payload[4:7], 'big')
        count = int.from_bytes(payload[10:14], 'big')
        print(f"Pkt {i}: UPDATE_BITMAP size={size_field} count={count}")
        
        # Collect the bitmap data from this and following packets
        update_data = bytearray()
        update_data.extend(payload[14:])  # rest of first chunk after header
        
        for j in range(i+1, i+100):
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
            # Stop at QUERY_STATUS
            if pay[0] == 0xcf:
                break
            update_data.extend(pay[:dl])
        
        print(f"  Collected {len(update_data)} bytes (size_field says {size_field})")
        
        # The data is size_field bytes of pixel data
        # Try to figure out dimensions
        # BGRA: size / 4 = pixels
        pixels = size_field // 4
        print(f"  Pixels (if BGRA): {pixels}")
        # Common dimensions for sensor regions
        for w in [184, 200, 150, 100, 50]:
            if pixels % w == 0:
                h = pixels // w
                if 10 <= h <= 100:
                    print(f"  Possible: {w}x{h}")
                    try:
                        img = Image.frombytes('RGBA', (w, h), bytes(update_data[:size_field]), 'raw', 'BGRA')
                        img.save(f'Debug/sensor_update_{w}x{h}.png')
                        print(f"  Saved Debug/sensor_update_{w}x{h}.png")
                    except:
                        pass
        break  # Just extract the first one
