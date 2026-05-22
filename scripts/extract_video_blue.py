#!/usr/bin/env python3
"""
Extract the uploaded video from BlueTheme-Flip180.txt PCAP data.
"""

import os
import sys

debug_dir = '/home/alex/GolandProjects/turing-screen/Debug/'
output_file = '/home/alex/GolandProjects/turing-screen/res/themes/NZXT_C/meilan_180.mp4'

if len(sys.argv) > 1:
    output_file = sys.argv[1]

content = open(os.path.join(debug_dir, 'BlueTheme-Flip180.txt')).read()

# Parse entries (multi-entry per line format)
entries = []
for line in content.strip().split('\n'):
    parts = line.split('\t')
    i = 0
    while i < len(parts):
        part = parts[i].strip()
        if part in ('0x00', '0x01'):
            direction = part
            i += 1
            while i < len(parts) and parts[i].strip() == '':
                i += 1
            if i < len(parts):
                hex_data = parts[i].strip()
                if hex_data:
                    entries.append((direction, hex_data))
            i += 1
        else:
            i += 1

# Entry 56: CREATE_FILE -> "create_success" at entry 57
# Entry 58-76: data chunks
# Entry 77: IN "file_rev_done img_show_"
# File size: 1081578

file_size = 1081578

print(f"Extracting video from BlueTheme-Flip180.txt")
print(f"Expected size: {file_size} bytes")
print(f"Output: {output_file}")
print()

data = bytearray()
for idx in range(58, 77):  # entries 58 to 76 are data
    direction, hex_data = entries[idx]
    if direction != '0x00':
        continue
    chunk = bytes.fromhex(hex_data)
    data.extend(chunk)
    print(f"  entry {idx:2d}: {len(chunk):>7} bytes (total: {len(data)})")

# Trim to exact file size
data = data[:file_size]

print(f"\nTotal extracted: {len(data)} bytes")
print(f"Expected:        {file_size} bytes")
print(f"Match: {len(data) == file_size}")

# Verify MP4 signature
if len(data) >= 8 and data[4:8] == b'ftyp':
    print("MP4 signature: OK (ftyp)")
else:
    print(f"WARNING: First 8 bytes: {data[:8].hex()}")

with open(output_file, 'wb') as f:
    f.write(data)

print(f"\nSaved to: {output_file}")
print(f"Size: {os.path.getsize(output_file)} bytes")
print(f"Try: mpv {output_file}")
