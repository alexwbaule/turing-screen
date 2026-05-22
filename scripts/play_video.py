#!/usr/bin/env python3
"""
Play a video on Turing Smart Screen 5" (Rev C).
The video must already exist on the device storage.

Sequence (validated from NZXTDinamicTheme.pcapng, gundam.pcapng):
  HELLO -> init
  STOP_VIDEO
  STOP_MEDIA -> "media_stop"
  SET_BRIGHTNESS
  GET_STORAGE_STATUS
  GET_FILE_INFO -> verify file exists
  RESTART_DEVICE
  (wait 2s)
  HELLO -> re-establish
  GET_FILE_INFO -> verify again
  PLAY_VIDEO -> "play_video_success"

Usage: python3 play_video.py [/root/video/earth.mp4]
"""

import sys
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice


def main():
    video_path = "/root/video/earth.mp4"
    if len(sys.argv) > 1:
        video_path = sys.argv[1]

    print(f"=== Play Video: {video_path} ===\n")

    dev = TuringDevice()
    try:
        dev.init()

        # GET_STORAGE_STATUS (as seen in all PCAPs)
        info = dev.get_storage_status()
        if info:
            total, used, free = info
            print(f"    Storage: {total} KB total, {used} KB used, {free} KB free")

        # GET_FILE_INFO — verify file exists
        size = dev.get_file_info(video_path)
        if size == 0:
            print(f"\n  ERROR: File not found on device: {video_path}")
            print(f"  Upload it first with: python3 upload_video.py <local_file>")
            sys.exit(1)
        print(f"    File size: {size} bytes ({size/1024:.1f} KB)")

        # RESTART_DEVICE + HELLO
        print()
        dev.restart_device(wait=2.0)

        # GET_FILE_INFO again (as seen in PCAP)
        size = dev.get_file_info(video_path)
        print(f"    File size: {size} bytes")

        # PLAY_VIDEO
        resp = dev.play_video(video_path, loop=True)

        print(f"\n=== {'SUCCESS' if resp == 'play_video_success' else 'FAILED: ' + str(resp)} ===")

    finally:
        dev.close()


if __name__ == "__main__":
    main()
