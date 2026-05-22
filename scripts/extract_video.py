#!/usr/bin/env python3
"""
Extract the uploaded video from SuperItem.txt PCAP data.
The video data is between CREATE_FILE response and file_rev_done response.
"""

import os
import sys

debug_dir = '/home/alex/GolandProjects/turing-screen/Debug/'
output_file = 'extracted_meilan_180.mp4'

if len(sys.argv) > 1:
    output_file = sys.argv[1]

lines = open(os.path.join(debug_dir, 'SuperItem.txt')).read().strip().split('\n')

# Lines 2218-2236 are the upload data (between "create_success" and "file_rev_done")
# File size from CREATE_FILE command: 1081578 bytes
file_size = 1081578

print(f"Extracting video from SuperItem.txt")
print(f"Expected size: {file_size} bytes")
print(f"Output: {output_file}")
print()

data = bytearray()
for i in range(2218, 2237):
    line = lines[i]
    parts = line.split('\t')
    if len(parts) < 3:
        continue
    if parts[0] != '0x00':  # Only OUT data
        continue
    hex_data = parts[2].strip()
    chunk = bytes.fromhex(hex_data)
    data.extend(chunk)
    print(f"  chunk {i-2218:2d}: {len(chunk):>7} bytes (total: {len(data)})")

# Trim to exact file size (last chunk has padding)
data = data[:file_size]

print(f"\nTotal extracted: {len(data)} bytes")
print(f"Expected:        {file_size} bytes")
print(f"Match: {len(data) == file_size}")

# Verify MP4 signature
if data[:4] == b'\x00\x00\x00\x20' and data[4:8] == b'ftyp':
    print("MP4 signature: OK (ftyp)")
else:
    print(f"WARNING: No MP4 signature found! First 8 bytes: {data[:8].hex()}")

with open(output_file, 'wb') as f:
    f.write(data)

print(f"\nSaved to: {output_file}")
print(f"Try: mpv {output_file}")
