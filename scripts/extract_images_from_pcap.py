#!/usr/bin/env python3
"""
Extract ALL images (overlay + sensor updates) from upload_full.pcapng.
Decodes the exact pixel format used by the device.
"""

import struct
import os
from PIL import Image

filepath = 'Debug/upload_full.pcapng'
with open(filepath, 'rb') as f:
    data = f.read()

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

os.makedirs('Debug/extracted_images', exist_ok=True)

# Helper: collect bulk OUT data from a range of packets
def collect_out_data(start, end):
    result = bytearray()
    for i in range(start, min(end, len(packets))):
        pkt = packets[i]
        if len(pkt) < 64:
            continue
        event = pkt[8]
        xfer_type = pkt[9]
        endpoint = pkt[10]
        data_len = struct.unpack_from('<I', pkt, 36)[0]
        payload = pkt[64:]
        if xfer_type == 3 and event == 0x53 and not (endpoint & 0x80) and data_len > 0:
            result.extend(payload[:data_len])
    return result


# ===== EXTRACT OVERLAY (0xca) =====
# After PLAY_VIDEO (pkt 8348), find the overlay sequence
print("\n=== OVERLAY EXTRACTION ===")

# Find PRE_UPDATE_BITMAP (0x86), SEPARATOR (0x2c), then SEND_OVERLAY (0xca)
overlay_start = None
overlay_end = None

for i in range(8350, 8700):
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
    
    if payload[0] == 0xca and overlay_start is None:
        overlay_start = i
        print(f"  SEND_OVERLAY starts at pkt {i}")
        print(f"  Header: {payload[:8].hex()}")
    
    # End at seq_png_init_sucess (bulk IN response)
    if overlay_start and xfer_type == 3 and event == 0x43 and (endpoint & 0x80) and len(payload) > 0:
        txt = payload.decode('ascii', errors='replace').rstrip('\x00')
        if 'seq_png' in txt:
            overlay_end = i
            print(f"  seq_png_init_sucess at pkt {i}")
            break

if overlay_start and overlay_end:
    # Collect all OUT data between overlay_start and overlay_end
    overlay_raw = bytearray()
    first_chunk = True
    for i in range(overlay_start, overlay_end):
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
        if data_len == 0:
            continue
        
        if first_chunk:
            # First chunk is the 0xca header — skip the header bytes (8 bytes)
            # But it's padded to 250 bytes, so the actual bitmap starts in next chunks
            # Actually the header is in a 250-byte padded chunk
            # Let's skip the first 250-byte chunk (the ca ef 69... header)
            first_chunk = False
            # The payload after the 8-byte header in this 250-byte chunk is bitmap data
            overlay_raw.extend(payload[8:data_len])
            continue
        
        overlay_raw.extend(payload[:data_len])
    
    print(f"  Raw overlay data: {len(overlay_raw)} bytes")
    print(f"  800*480*4 (BGRA) = {800*480*4}")
    print(f"  800*480*3 (BGR) = {800*480*3}")
    
    # The overlay format from GenerateBackgroundImage is BGRA (4 bytes per pixel)
    # But the PCAP shows it's sent via the 0xca command which uses the same
    # format as 0xc8 (SendPayload) — let's check both
    
    # Try BGRA
    if len(overlay_raw) >= 800*480*4:
        try:
            img = Image.frombytes('RGBA', (800, 480), bytes(overlay_raw[:800*480*4]), 'raw', 'BGRA')
            img.save('Debug/extracted_images/overlay_BGRA.png')
            print(f"  Saved overlay_BGRA.png")
        except Exception as e:
            print(f"  BGRA failed: {e}")
    
    # Try BGR
    if len(overlay_raw) >= 800*480*3:
        try:
            img = Image.frombytes('RGB', (800, 480), bytes(overlay_raw[:800*480*3]), 'raw', 'BGR')
            img.save('Debug/extracted_images/overlay_BGR.png')
            print(f"  Saved overlay_BGR.png")
        except Exception as e:
            print(f"  BGR failed: {e}")
    
    # Also try skipping different header sizes
    for skip in [0, 242, 250, 500]:
        trimmed = overlay_raw[skip:]
        if len(trimmed) >= 800*480*4:
            try:
                img = Image.frombytes('RGBA', (800, 480), bytes(trimmed[:800*480*4]), 'raw', 'BGRA')
                img.save(f'Debug/extracted_images/overlay_BGRA_skip{skip}.png')
                print(f"  Saved overlay_BGRA_skip{skip}.png")
            except:
                pass


# ===== EXTRACT SENSOR UPDATES (0xcc) =====
print("\n=== SENSOR UPDATE EXTRACTION ===")

# Find UPDATE_BITMAP commands after the overlay
update_count = 0
for i in range(8600, min(9000, len(packets))):
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
    if payload[0] != 0xcc:
        continue
    
    # UPDATE_BITMAP header: cc ef 69 00 [size:3BE] [pad:3] [count:4BE]
    size_field = int.from_bytes(payload[4:7], 'big')
    count = int.from_bytes(payload[10:14], 'big')
    
    # Collect all data for this update (until QUERY_STATUS 0xcf)
    update_data = bytearray()
    # First chunk: after the 14-byte header, rest of this 250-byte chunk is data
    update_data.extend(payload[14:data_len])
    
    for j in range(i+1, i+200):
        if j >= len(packets):
            break
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
        if pay[0] == 0xcf:  # QUERY_STATUS = end
            break
        update_data.extend(pay[:dl])
    
    # Parse the update data format:
    # Each line: [position:3BE][width:2BE][pixel_data: width * bytes_per_pixel]
    # Ends with: ef 69
    
    # Determine encoding by checking size vs expected
    # Try to decode the position/width format
    print(f"\n  Update #{update_count} (pkt {i}): size_field={size_field}, count={count}, collected={len(update_data)}")
    
    # Parse line by line
    off = 0
    lines = []
    min_x = 9999
    max_x = 0
    min_y = 9999
    max_y = 0
    bpp = 0  # bytes per pixel (detect)
    
    while off < min(size_field, len(update_data)) - 2:
        if update_data[off] == 0xef and off + 1 < len(update_data) and update_data[off+1] == 0x69:
            break  # End marker
        
        if off + 5 > len(update_data):
            break
        
        position = int.from_bytes(update_data[off:off+3], 'big')
        width = int.from_bytes(update_data[off+3:off+5], 'big')
        off += 5
        
        if width == 0 or width > 800:
            break
        
        # Calculate y and x from position (position = y * 800 + x)
        y = position // 800
        x = position % 800
        
        if y > 480 or x > 800:
            break
        
        min_x = min(min_x, x)
        max_x = max(max_x, x + width)
        min_y = min(min_y, y)
        max_y = max(max_y, y)
        
        # Determine bytes per pixel from remaining data
        if bpp == 0:
            # Try 3 (BGR) and 4 (BGRA)
            remaining = min(size_field, len(update_data)) - off
            # If we assume this is one line of `width` pixels:
            if remaining >= width * 4:
                bpp = 4  # Try BGRA first
            elif remaining >= width * 3:
                bpp = 3
            else:
                break
        
        pixel_bytes = width * bpp
        if off + pixel_bytes > len(update_data):
            break
        
        lines.append((x, y, width, update_data[off:off+pixel_bytes]))
        off += pixel_bytes
    
    if lines:
        h = max_y - min_y + 1
        w = max_x - min_x
        print(f"    Region: x={min_x}, y={min_y}, w={w}, h={h}, bpp={bpp}, lines={len(lines)}")
        
        # Reconstruct image
        if w > 0 and h > 0 and bpp > 0:
            if bpp == 4:
                img = Image.new('RGBA', (w, h), (0, 0, 0, 0))
            else:
                img = Image.new('RGB', (w, h), (0, 0, 0))
            
            for lx, ly, lw, lpixels in lines:
                for px in range(lw):
                    if bpp == 4:
                        b, g, r, a = lpixels[px*4], lpixels[px*4+1], lpixels[px*4+2], lpixels[px*4+3]
                        img.putpixel((lx - min_x + px, ly - min_y), (r, g, b, a))
                    else:
                        b, g, r = lpixels[px*3], lpixels[px*3+1], lpixels[px*3+2]
                        img.putpixel((lx - min_x + px, ly - min_y), (r, g, b))
            
            fname = f'Debug/extracted_images/sensor_update_{update_count}_{w}x{h}_at_{min_x}_{min_y}.png'
            img.save(fname)
            print(f"    Saved {fname}")
    
    update_count += 1
    if update_count >= 5:
        break

print(f"\nExtracted {update_count} sensor updates")
