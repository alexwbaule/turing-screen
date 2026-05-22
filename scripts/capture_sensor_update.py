#!/usr/bin/env python3
"""
Capture a live sensor UPDATE_BITMAP from the Go binary and decode it as an image.
Connects to the device, sends HELLO, then listens for UPDATE_BITMAP data.

Actually simpler: let's just intercept what the Go binary sends by
running it and capturing the serial traffic. But that's complex.

Instead: let's decode the sensor updates from the upload_full.pcapng
using the EXACT format from GeneratePartialImage:
  Each line: [position:3BE][width:2BE][pixel_data: width * 3 bytes (BGR)]
  End marker: [0xef][0x69]
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

os.makedirs('Debug/extracted_images', exist_ok=True)

# Find UPDATE_BITMAP (0xcc) commands after the overlay
# The overlay ends with seq_png_init_sucess at pkt 8503
# Sensor updates start after that

print("=== Extracting sensor UPDATE_BITMAP images ===\n")

update_count = 0
for i in range(8600, min(9100, len(packets))):
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

    # UPDATE_BITMAP header (14 bytes in first 250-byte chunk):
    # cc ef 69 00 [size:3BE] [pad:3] [count:4BE]
    size_field = int.from_bytes(payload[4:7], 'big')
    count = int.from_bytes(payload[10:14], 'big')

    # Collect ALL data for this update from this and subsequent packets
    # The data is in 250-byte chunks. First chunk: 14 bytes header + 236 bytes data
    # But it's all in one USB transfer of data_len bytes
    
    # The image data starts at byte 250 (second 250-byte chunk)
    # because the first 250-byte chunk is the header padded with zeros
    if data_len < 500:
        continue  # Too small, skip
    
    img_data = payload[250:]  # Skip the first 250-byte header chunk
    
    # Also collect from subsequent packets if this update spans multiple USB transfers
    # (check if next packet is also bulk OUT and not a new command)
    j = i + 1
    while j < min(i + 50, len(packets)):
        p = packets[j]
        if len(p) < 64:
            j += 1
            continue
        ev = p[8]
        xf = p[9]
        ep = p[10]
        dl = struct.unpack_from('<I', p, 36)[0]
        pay = p[64:]
        if xf != 3 or ev != 0x53 or (ep & 0x80):
            break
        if dl == 0:
            j += 1
            continue
        # If it starts with a known command byte, it's a new command
        if pay[0] in (0xcc, 0xcf, 0x01, 0x79, 0x96):
            break
        img_data += pay[:dl]
        j += 1

    # Parse the image data format:
    # Each line: [position:3BE][width:2BE][BGR pixels: width*3 bytes]
    # End: ef 69
    
    lines = []
    off = 0
    bpp = 3  # BGR
    
    while off < min(size_field, len(img_data)) - 5:
        # Check for end marker
        if img_data[off] == 0xef and off + 1 < len(img_data) and img_data[off+1] == 0x69:
            break
        
        if off + 5 > len(img_data):
            break
        
        position = int.from_bytes(img_data[off:off+3], 'big')
        width = int.from_bytes(img_data[off+3:off+5], 'big')
        off += 5
        
        if width == 0 or width > 800:
            break
        
        # position = row * 800 + col (from the Go code: (x0+h)*display.Width + y0)
        row = position // 800
        col = position % 800
        
        if row > 480 or col + width > 800:
            break
        
        pixel_bytes = width * bpp
        if off + pixel_bytes > len(img_data):
            break
        
        lines.append((col, row, width, img_data[off:off+pixel_bytes]))
        off += pixel_bytes
    
    if len(lines) < 2:
        continue
    
    # Determine bounding box
    min_col = min(col for col, row, w, _ in lines)
    max_col = max(col + w for col, row, w, _ in lines)
    min_row = min(row for col, row, w, _ in lines)
    max_row = max(row for col, row, w, _ in lines)
    
    img_w = max_col - min_col
    img_h = max_row - min_row + 1
    
    if img_w <= 0 or img_h <= 0 or img_w > 800 or img_h > 480:
        continue
    
    # Create image
    img = Image.new('RGB', (img_w, img_h), (0, 0, 0))
    
    for col, row, width, pixels in lines:
        for px in range(width):
            if px * 3 + 2 < len(pixels):
                b, g, r = pixels[px*3], pixels[px*3+1], pixels[px*3+2]
                x = col - min_col + px
                y = row - min_row
                if 0 <= x < img_w and 0 <= y < img_h:
                    img.putpixel((x, y), (r, g, b))
    
    fname = f'Debug/extracted_images/update_{update_count:02d}_pkt{i}_{img_w}x{img_h}_at_{min_col}_{min_row}.png'
    img.save(fname)
    print(f"  #{update_count}: pkt {i}, {img_w}x{img_h} at ({min_col},{min_row}), {len(lines)} lines, size={size_field}")
    print(f"    -> {fname}")
    
    update_count += 1
    if update_count >= 10:
        break

print(f"\nExtracted {update_count} sensor update images")
