#!/usr/bin/env python3
"""
Deep analysis of upload_full.pcapng - extract the COMPLETE protocol
including all control transfers, interrupt endpoint activity, and timing.
"""

import struct
import sys

filepath = 'Debug/upload_full.pcapng'
with open(filepath, 'rb') as f:
    data = f.read()

# Parse legacy PCAP
offset = 24
packets = []
while offset < len(data):
    if offset + 16 > len(data):
        break
    ts_sec = struct.unpack_from('<I', data, offset)[0]
    ts_usec = struct.unpack_from('<I', data, offset + 4)[0]
    cap_len = struct.unpack_from('<I', data, offset + 8)[0]
    pkt_data = data[offset + 16: offset + 16 + cap_len]
    packets.append((ts_sec, ts_usec, pkt_data))
    offset += 16 + cap_len

print(f"Total packets: {len(packets)}")
print()

# Analyze the COMPLETE gunpla upload (last one, most recent)
# CREATE_FILE at pkt 7972, file_rev_done at pkt 8144
# But let's start from the DTR/RTS setup (around pkt 6480)

# First find where the gunpla session starts (the port open/DTR sequence)
print("=" * 70)
print("COMPLETE GUNPLA UPLOAD SESSION")
print("=" * 70)
print()

# Show EVERYTHING from the port open (DTR toggle) to file_rev_done
# The DTR toggle before gunpla upload is around pkt 6480
start_pkt = 6460
end_pkt = 8160

for i in range(start_pkt, min(end_pkt, len(packets))):
    ts_s, ts_us, pkt = packets[i]
    if len(pkt) < 64:
        continue
    
    event = pkt[8]
    xfer_type = pkt[9]
    endpoint = pkt[10]
    setup_flag = pkt[14]
    status = struct.unpack_from('<i', pkt, 28)[0]
    urb_len = struct.unpack_from('<I', pkt, 32)[0]
    data_len = struct.unpack_from('<I', pkt, 36)[0]
    payload = pkt[64:]
    
    ev = chr(event) if event in range(32, 127) else f'0x{event:02x}'
    xfer_names = {0: 'ISO', 1: 'INT', 2: 'CTRL', 3: 'BULK'}
    xfer = xfer_names.get(xfer_type, '?')
    direction = 'IN' if (endpoint & 0x80) else 'OUT'
    
    # Skip Complete events for OUT bulk (just ACKs, no data)
    if event == 0x43 and xfer_type == 3 and not (endpoint & 0x80) and data_len == 0:
        continue
    
    # Skip Submit events for IN bulk with no data (just read requests)
    if event == 0x53 and xfer_type == 3 and (endpoint & 0x80) and data_len == 0:
        continue
    
    # Skip interrupt endpoint noise (unless it has data or interesting status)
    if endpoint == 0x84 and data_len == 0 and status in (0, -115, -71):
        # Only show the first few and count the rest
        continue
    
    line = f"  {i:5d}: {ev} ep=0x{endpoint:02x}({direction}) {xfer} st={status} dl={data_len}"
    
    # Control transfers - show setup
    if xfer_type == 2 and event == 0x53 and setup_flag == 0:
        bmReqType = pkt[40]
        bReq = pkt[41]
        wVal = struct.unpack_from('<H', pkt, 42)[0]
        wIdx = struct.unpack_from('<H', pkt, 44)[0]
        wLen = struct.unpack_from('<H', pkt, 46)[0]
        
        if bReq == 0x20 and len(payload) >= 7:
            baud = struct.unpack_from('<I', payload, 0)[0]
            line = f"  {i:5d}: CTRL SET_LINE_CODING baud={baud} data={payload[6]} stop={payload[4]}"
        elif bReq == 0x22:
            line = f"  {i:5d}: CTRL SET_CONTROL_LINE_STATE DTR={wVal&1} RTS={(wVal>>1)&1}"
        elif bReq == 0x01:
            line = f"  {i:5d}: CTRL CLEAR_FEATURE ep=0x{wIdx:04x}"
        elif bReq == 0x09:
            line = f"  {i:5d}: CTRL SET_CONFIGURATION val={wVal}"
        else:
            line = f"  {i:5d}: CTRL bmReqType=0x{bmReqType:02x} bReq=0x{bReq:02x} wVal=0x{wVal:04x} wIdx=0x{wIdx:04x} wLen={wLen}"
        print(line)
        continue
    
    # Bulk OUT with data (commands or file data)
    if xfer_type == 3 and event == 0x53 and not (endpoint & 0x80) and data_len > 0:
        if data_len == 250:
            # Command
            cmd = payload[0]
            cmds = {0x01:'HELLO', 0x79:'STOP_VIDEO', 0x96:'STOP_MEDIA', 0x7b:'SET_BRIGHTNESS',
                    0x64:'GET_STORAGE', 0x6e:'GET_FILE_INFO', 0x65:'LIST_DIR',
                    0x66:'DELETE_FILE', 0x6f:'CREATE_FILE', 0x78:'PLAY_VIDEO',
                    0x82:'RESTART_DEVICE', 0x86:'PRE_UPDATE', 0x7d:'CMD_0x7D',
                    0xcf:'QUERY_STATUS'}
            name = cmds.get(cmd, f'CMD_0x{cmd:02x}')
            extra = ''
            if cmd == 0x6f and len(payload) >= 14:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10+plen+4 <= len(payload):
                    path = payload[10:10+plen].decode('ascii', errors='replace')
                    fsize = int.from_bytes(payload[10+plen:10+plen+4], 'little')
                    extra = f' "{path}" size={fsize}'
            elif cmd == 0x6e and len(payload) >= 10:
                plen = int.from_bytes(payload[3:7], 'big')
                if 10+plen <= len(payload):
                    extra = f' "{payload[10:10+plen].decode("ascii", errors="replace")}"'
            elif cmd == 0x7d:
                extra = f' data={payload[:12].hex()}'
            elif cmd == 0x7b and len(payload) >= 11:
                extra = f' ({payload[10]})'
            print(f"  {i:5d}: BULK OUT {name}{extra}")
        else:
            # File data
            print(f"  {i:5d}: BULK OUT DATA {data_len} bytes (first4={payload[:4].hex()}, last4={payload[-4:].hex() if len(payload)>=4 else ''})")
        continue
    
    # Bulk IN with response
    if xfer_type == 3 and event == 0x43 and (endpoint & 0x80) and data_len > 0 and len(payload) > 0:
        txt = payload[:50].decode('ascii', errors='replace').rstrip('\x00')
        if txt:
            print(f"  {i:5d}: BULK IN  \"{txt}\"")
        continue
    
    # Control transfer complete (show only non-zero status)
    if xfer_type == 2 and event == 0x43:
        if status != 0:
            print(f"  {i:5d}: CTRL COMPLETE status={status}")
        continue


# Now count interrupt endpoint activity between last data and file_rev_done
print()
print("=" * 70)
print("INTERRUPT ENDPOINT ACTIVITY (between last data write and file_rev_done)")
print("=" * 70)

# Last data write is pkt 8054, file_rev_done is pkt 8144
int_count = 0
ctrl_count = 0
for i in range(8054, 8145):
    ts_s, ts_us, pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer_type = pkt[9]
    endpoint = pkt[10]
    status = struct.unpack_from('<i', pkt, 28)[0]
    
    if endpoint == 0x84:
        int_count += 1
    if xfer_type == 2:
        ctrl_count += 1

print(f"  Interrupt EP 0x84 events: {int_count}")
print(f"  Control transfers: {ctrl_count}")
print(f"  Total packets in gap: {8144 - 8054}")

# Show the EXACT control transfers in this gap
print()
print("  Detail:")
for i in range(8054, 8145):
    ts_s, ts_us, pkt = packets[i]
    if len(pkt) < 64:
        continue
    event = pkt[8]
    xfer_type = pkt[9]
    endpoint = pkt[10]
    setup_flag = pkt[14]
    status = struct.unpack_from('<i', pkt, 28)[0]
    data_len = struct.unpack_from('<I', pkt, 36)[0]
    payload = pkt[64:]
    
    ev = chr(event) if event in range(32, 127) else f'0x{event:02x}'
    
    if xfer_type == 2 and event == 0x53 and setup_flag == 0:
        bReq = pkt[41]
        wVal = struct.unpack_from('<H', pkt, 42)[0]
        wIdx = struct.unpack_from('<H', pkt, 44)[0]
        print(f"    {i}: S CTRL bReq=0x{bReq:02x} wVal=0x{wVal:04x} wIdx=0x{wIdx:04x}")
    elif endpoint == 0x84:
        print(f"    {i}: {ev} INT ep=0x84 st={status} dl={data_len}")
    elif xfer_type == 3 and event == 0x43 and (endpoint & 0x80) and len(payload) > 0:
        txt = payload[:30].decode('ascii', errors='replace').rstrip('\x00')
        if txt:
            print(f"    {i}: C BULK IN \"{txt}\"")
