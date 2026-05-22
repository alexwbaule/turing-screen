#!/usr/bin/env python3
"""Analyze TURZX.exe for upload-related constants and strings."""

import struct

with open('FromApp/5inchENG/TURZX-V3.1.0-5inchENG/TURZX.exe', 'rb') as f:
    data = f.read()

# Search for key strings (UTF-16LE)
searches = ['transfer speed', 'create_success', 'file_rev_done', 'WriteBufferSize',
            'ReadBufferSize', 'WritePipe', 'BulkTransfer', 'CMD_INTERRUPT']
for s in searches:
    needle = s.encode('utf-16-le')
    idx = data.find(needle)
    if idx >= 0:
        print(f'Found "{s}" at offset 0x{idx:x}')

print()

# Search for common buffer/chunk size constants as int32 LE
print("Numeric constants (int32 LE):")
for val in [1048576, 524288, 262144, 131072, 65536, 32768, 25000, 16384, 8192, 4096, 2048, 1024, 512, 250]:
    needle = struct.pack('<I', val)
    count = data.count(needle)
    if 0 < count < 50:
        # Find first few occurrences
        positions = []
        start = 0
        for _ in range(min(count, 5)):
            idx = data.find(needle, start)
            if idx < 0:
                break
            positions.append(f'0x{idx:x}')
            start = idx + 1
        print(f'  {val:>10} (0x{val:06x}): {count:3d} times  first: {", ".join(positions)}')

print()

# Look for the SerialPortStream configuration
# Search for baud rate 115200 = 0x1C200
needle = struct.pack('<I', 115200)
count = data.count(needle)
print(f'Baud 115200: {count} occurrences')

# Search for 921600 (high speed serial)
needle = struct.pack('<I', 921600)
count = data.count(needle)
print(f'Baud 921600: {count} occurrences')

# Search for WriteTimeout values
for val in [5000, 10000, 3000, 2000, 1000, 500, 100]:
    needle = struct.pack('<I', val)
    count = data.count(needle)
    if 0 < count < 30:
        print(f'  Timeout/value {val}: {count} occurrences')
