#!/usr/bin/env python3
"""
Extracts content from .turtheme files (Turing Screen themes).

.turtheme files are .NET BinaryFormatter serialized objects containing
UsbMonitorL.Theme with embedded images (PNG), configuration, and references.

This script properly parses the NRBF (.NET Binary Remoting Format) to extract:
- Theme metadata (name, resolution, orientation, video path)
- Embedded images (PNG, JPEG, GIF)
- GraphItem data (sensor type, position, font config, colors)
- Status bar and radial/arch bar configurations

Usage:
    python3 extract_turtheme.py <file_or_directory> [--output-dir DIR] [--all]
"""

import hashlib
import os
import sys
import struct
import yaml
import shutil
from pathlib import Path
from typing import Optional, Any


# ---------------------------------------------------------------------------
# NRBF Constants
# ---------------------------------------------------------------------------

# Record Types
RT_SERIALIZED_STREAM_HEADER = 0x00
RT_CLASS_WITH_ID = 0x01
RT_SYSTEM_CLASS_WITH_MEMBERS_AND_TYPES = 0x04
RT_CLASS_WITH_MEMBERS_AND_TYPES = 0x05
RT_BINARY_OBJECT_STRING = 0x06
RT_BINARY_ARRAY = 0x07
RT_MEMBER_PRIMITIVE_TYPED = 0x08
RT_MEMBER_REFERENCE = 0x09
RT_OBJECT_NULL = 0x0A
RT_MESSAGE_END = 0x0B
RT_BINARY_LIBRARY = 0x0C
RT_OBJECT_NULL_MULTIPLE_256 = 0x0D
RT_OBJECT_NULL_MULTIPLE = 0x0E
RT_ARRAY_SINGLE_PRIMITIVE = 0x0F
RT_ARRAY_SINGLE_OBJECT = 0x10
RT_ARRAY_SINGLE_STRING = 0x11

# Primitive Type Enums
PT_BOOLEAN = 1
PT_BYTE = 2
PT_CHAR = 3
PT_DECIMAL = 5
PT_DOUBLE = 6
PT_INT16 = 7
PT_INT32 = 8
PT_INT64 = 9
PT_SBYTE = 10
PT_SINGLE = 11
PT_TIMESPAN = 12
PT_DATETIME = 13
PT_UINT16 = 14
PT_UINT32 = 15
PT_UINT64 = 16
PT_NULL = 17
PT_STRING = 18

# Binary Type Enums
BT_PRIMITIVE = 0
BT_STRING = 1
BT_OBJECT = 2
BT_SYSTEM_CLASS = 3
BT_CLASS = 4
BT_OBJECT_ARRAY = 5
BT_STRING_ARRAY = 6
BT_PRIMITIVE_ARRAY = 7

PRIMITIVE_SIZES = {
    PT_BOOLEAN: 1, PT_BYTE: 1, PT_CHAR: 1, PT_SBYTE: 1,
    PT_INT16: 2, PT_UINT16: 2,
    PT_INT32: 4, PT_UINT32: 4, PT_SINGLE: 4,
    PT_INT64: 8, PT_UINT64: 8, PT_DOUBLE: 8,
    PT_DATETIME: 8, PT_TIMESPAN: 8,
}


# ---------------------------------------------------------------------------
# NRBF Parser
# ---------------------------------------------------------------------------

def read_lps(data: bytes, pos: int) -> tuple[Optional[str], int]:
    """Read .NET LengthPrefixedString (7-bit encoded length)."""
    if pos >= len(data):
        return None, pos
    length = 0
    shift = 0
    while True:
        if pos >= len(data):
            return None, pos
        b = data[pos]; pos += 1
        length |= (b & 0x7F) << shift
        if b & 0x80 == 0:
            break
        shift += 7
        if shift >= 35:
            break
    if pos + length > len(data):
        return None, pos
    s = data[pos:pos + length]
    try:
        return s.decode("utf-8", errors="replace"), pos + length
    except Exception:
        return s.decode("latin-1", errors="replace"), pos + length


def read_int32(data, pos):
    return struct.unpack_from("<i", data, pos)[0], pos + 4

def read_uint32(data, pos):
    return struct.unpack_from("<I", data, pos)[0], pos + 4

def read_int16(data, pos):
    return struct.unpack_from("<h", data, pos)[0], pos + 2

def read_byte(data, pos):
    return data[pos], pos + 1

def read_single(data, pos):
    return struct.unpack_from("<f", data, pos)[0], pos + 4

def read_double(data, pos):
    return struct.unpack_from("<d", data, pos)[0], pos + 8


class NRBFParser:
    """Parser for .NET Binary Remoting Format (NRBF) streams."""

    def __init__(self, data: bytes):
        self.data = data
        self.pos = 0
        self.objects: dict[int, Any] = {}
        self.class_defs: dict[int, dict] = {}
        self.libraries: dict[int, str] = {}

    def parse(self, max_records=100000):
        """Parse the entire NRBF stream."""
        records = 0
        while self.pos < len(self.data) and records < max_records:
            start_pos = self.pos
            try:
                rt = self.data[self.pos]
                if rt == RT_SERIALIZED_STREAM_HEADER:
                    self._parse_header()
                elif rt == RT_BINARY_LIBRARY:
                    self._parse_library()
                elif rt == RT_CLASS_WITH_MEMBERS_AND_TYPES:
                    self._parse_class_with_members_and_types()
                elif rt == RT_SYSTEM_CLASS_WITH_MEMBERS_AND_TYPES:
                    self._parse_system_class_with_members_and_types()
                elif rt == RT_CLASS_WITH_ID:
                    self._parse_class_with_id()
                elif rt == RT_BINARY_OBJECT_STRING:
                    self._parse_binary_object_string()
                elif rt == RT_MEMBER_REFERENCE:
                    self.pos += 5
                elif rt == RT_OBJECT_NULL:
                    self.pos += 1
                elif rt == RT_OBJECT_NULL_MULTIPLE_256:
                    self.pos += 2
                elif rt == RT_OBJECT_NULL_MULTIPLE:
                    self.pos += 5
                elif rt == RT_BINARY_ARRAY:
                    self._parse_binary_array()
                elif rt == RT_ARRAY_SINGLE_PRIMITIVE:
                    self._parse_array_single_primitive()
                elif rt == RT_ARRAY_SINGLE_OBJECT:
                    self._parse_array_single_object()
                elif rt == RT_MEMBER_PRIMITIVE_TYPED:
                    self._parse_member_primitive_typed()
                elif rt == RT_MESSAGE_END:
                    self.pos += 1
                    break
                else:
                    self.pos += 1
            except (struct.error, IndexError, OverflowError):
                break
            if self.pos == start_pos:
                self.pos += 1
            records += 1

    def _parse_header(self):
        self.pos += 1 + 4 + 4 + 4 + 4  # RecordType + rootId + headerId + major + minor

    def _parse_library(self):
        self.pos += 1
        lib_id, self.pos = read_int32(self.data, self.pos)
        lib_name, self.pos = read_lps(self.data, self.pos)
        self.libraries[lib_id] = lib_name

    def _parse_class_with_members_and_types(self):
        self.pos += 1
        object_id, self.pos = read_int32(self.data, self.pos)
        class_name, self.pos = read_lps(self.data, self.pos)
        member_count, self.pos = read_int32(self.data, self.pos)
        if member_count < 0 or member_count > 200:
            return  # Corrupt data, bail
        member_names = []
        for _ in range(member_count):
            name, self.pos = read_lps(self.data, self.pos)
            member_names.append(name)
        member_types = []
        for _ in range(member_count):
            bt, self.pos = read_byte(self.data, self.pos)
            member_types.append(bt)
        additional_info = self._read_additional_info(member_types)
        lib_id, self.pos = read_int32(self.data, self.pos)
        class_def = {
            "class_name": class_name, "member_names": member_names,
            "member_types": member_types, "additional_info": additional_info,
        }
        self.class_defs[object_id] = class_def
        values = self._read_member_values(class_def)
        obj = {"_class": class_name, "_id": object_id}
        for n, v in zip(member_names, values):
            obj[n] = v
        self.objects[object_id] = obj

    def _parse_system_class_with_members_and_types(self):
        self.pos += 1
        object_id, self.pos = read_int32(self.data, self.pos)
        class_name, self.pos = read_lps(self.data, self.pos)
        member_count, self.pos = read_int32(self.data, self.pos)
        if member_count < 0 or member_count > 200:
            return  # Corrupt data, bail
        member_names = []
        for _ in range(member_count):
            name, self.pos = read_lps(self.data, self.pos)
            member_names.append(name)
        member_types = []
        for _ in range(member_count):
            bt, self.pos = read_byte(self.data, self.pos)
            member_types.append(bt)
        additional_info = self._read_additional_info(member_types)
        class_def = {
            "class_name": class_name, "member_names": member_names,
            "member_types": member_types, "additional_info": additional_info,
        }
        self.class_defs[object_id] = class_def
        values = self._read_member_values(class_def)
        obj = {"_class": class_name, "_id": object_id}
        for n, v in zip(member_names, values):
            obj[n] = v
        self.objects[object_id] = obj

    def _parse_class_with_id(self):
        self.pos += 1
        object_id, self.pos = read_int32(self.data, self.pos)
        ref_id, self.pos = read_int32(self.data, self.pos)
        if ref_id not in self.class_defs:
            return
        class_def = self.class_defs[ref_id]
        self.class_defs[object_id] = class_def
        values = self._read_member_values(class_def)
        obj = {"_class": class_def["class_name"], "_id": object_id}
        for n, v in zip(class_def["member_names"], values):
            obj[n] = v
        self.objects[object_id] = obj

    def _parse_binary_object_string(self):
        self.pos += 1
        object_id, self.pos = read_int32(self.data, self.pos)
        value, self.pos = read_lps(self.data, self.pos)
        self.objects[object_id] = value  # Store as plain string

    def _parse_binary_array(self):
        self.pos += 1
        object_id, self.pos = read_int32(self.data, self.pos)
        array_type, self.pos = read_byte(self.data, self.pos)
        rank, self.pos = read_int32(self.data, self.pos)
        lengths = []
        for _ in range(rank):
            l, self.pos = read_int32(self.data, self.pos)
            lengths.append(l)
        if array_type in (3, 4, 5):
            for _ in range(rank):
                _, self.pos = read_int32(self.data, self.pos)
        bt, self.pos = read_byte(self.data, self.pos)
        if bt == BT_PRIMITIVE:
            pt, self.pos = read_byte(self.data, self.pos)
            total = 1
            for l in lengths:
                total *= l
            self.pos += PRIMITIVE_SIZES.get(pt, 4) * total
        elif bt == BT_CLASS:
            _, self.pos = read_lps(self.data, self.pos)
            _, self.pos = read_int32(self.data, self.pos)
        elif bt == BT_SYSTEM_CLASS:
            _, self.pos = read_lps(self.data, self.pos)

    def _parse_array_single_primitive(self):
        self.pos += 1
        _, self.pos = read_int32(self.data, self.pos)
        length, self.pos = read_int32(self.data, self.pos)
        pt, self.pos = read_byte(self.data, self.pos)
        self.pos += PRIMITIVE_SIZES.get(pt, 4) * length

    def _parse_array_single_object(self):
        self.pos += 1
        _, self.pos = read_int32(self.data, self.pos)
        _, self.pos = read_int32(self.data, self.pos)

    def _parse_member_primitive_typed(self):
        self.pos += 1
        pt, self.pos = read_byte(self.data, self.pos)
        self.pos += PRIMITIVE_SIZES.get(pt, 4)

    def _read_additional_info(self, member_types):
        additional_info = []
        for bt in member_types:
            if bt == BT_PRIMITIVE:
                pt, self.pos = read_byte(self.data, self.pos)
                additional_info.append(("primitive", pt))
            elif bt == BT_SYSTEM_CLASS:
                cn, self.pos = read_lps(self.data, self.pos)
                additional_info.append(("system_class", cn))
            elif bt == BT_CLASS:
                cn, self.pos = read_lps(self.data, self.pos)
                lid, self.pos = read_int32(self.data, self.pos)
                additional_info.append(("class", cn, lid))
            elif bt == BT_PRIMITIVE_ARRAY:
                pt, self.pos = read_byte(self.data, self.pos)
                additional_info.append(("primitive_array", pt))
            else:
                additional_info.append((bt,))
        return additional_info

    def _read_member_values(self, class_def):
        values = []
        for bt, ai in zip(class_def["member_types"], class_def["additional_info"]):
            if bt == BT_PRIMITIVE:
                val = self._read_primitive(ai[1])
                values.append(val)
            else:
                val = self._read_inline_value()
                values.append(val)
        return values

    def _read_primitive(self, pt):
        if pt == PT_BOOLEAN:
            val, self.pos = read_byte(self.data, self.pos)
            return bool(val)
        elif pt == PT_BYTE:
            val, self.pos = read_byte(self.data, self.pos)
            return val
        elif pt == PT_INT16:
            val, self.pos = read_int16(self.data, self.pos)
            return val
        elif pt == PT_INT32:
            val, self.pos = read_int32(self.data, self.pos)
            return val
        elif pt == PT_INT64:
            val = struct.unpack_from("<q", self.data, self.pos)[0]
            self.pos += 8
            return val
        elif pt == PT_SINGLE:
            val, self.pos = read_single(self.data, self.pos)
            return val
        elif pt == PT_DOUBLE:
            val, self.pos = read_double(self.data, self.pos)
            return val
        elif pt == PT_UINT16:
            val = struct.unpack_from("<H", self.data, self.pos)[0]
            self.pos += 2
            return val
        elif pt == PT_UINT32:
            val, self.pos = read_uint32(self.data, self.pos)
            return val
        elif pt == PT_UINT64:
            val = struct.unpack_from("<Q", self.data, self.pos)[0]
            self.pos += 8
            return val
        elif pt == PT_CHAR:
            val, self.pos = read_byte(self.data, self.pos)
            return chr(val)
        elif pt in (PT_DATETIME, PT_TIMESPAN):
            self.pos += 8
            return None
        else:
            return None

    def _read_inline_value(self):
        """Read an inline record value."""
        if self.pos >= len(self.data):
            return None
        rt = self.data[self.pos]
        if rt == RT_BINARY_OBJECT_STRING:
            self.pos += 1
            oid, self.pos = read_int32(self.data, self.pos)
            val, self.pos = read_lps(self.data, self.pos)
            self.objects[oid] = val
            return val
        elif rt == RT_OBJECT_NULL:
            self.pos += 1
            return None
        elif rt == RT_MEMBER_REFERENCE:
            self.pos += 1
            ref_id, self.pos = read_int32(self.data, self.pos)
            return ("__ref__", ref_id)
        elif rt == RT_CLASS_WITH_MEMBERS_AND_TYPES:
            # Peek at the object_id before parsing
            oid = struct.unpack_from("<i", self.data, self.pos + 1)[0]
            self._parse_class_with_members_and_types()
            return ("__ref__", oid)
        elif rt == RT_CLASS_WITH_ID:
            oid = struct.unpack_from("<i", self.data, self.pos + 1)[0]
            self._parse_class_with_id()
            return ("__ref__", oid)
        elif rt == RT_SYSTEM_CLASS_WITH_MEMBERS_AND_TYPES:
            oid = struct.unpack_from("<i", self.data, self.pos + 1)[0]
            self._parse_system_class_with_members_and_types()
            return ("__ref__", oid)
        elif rt == RT_BINARY_ARRAY:
            self._parse_binary_array()
            return ("__array__",)
        elif rt == RT_ARRAY_SINGLE_PRIMITIVE:
            self._parse_array_single_primitive()
            return ("__prim_array__",)
        elif rt == RT_ARRAY_SINGLE_OBJECT:
            self._parse_array_single_object()
            return ("__obj_array__",)
        elif rt == RT_OBJECT_NULL_MULTIPLE_256:
            self.pos += 1
            _, self.pos = read_byte(self.data, self.pos)
            return None
        elif rt == RT_OBJECT_NULL_MULTIPLE:
            self.pos += 1
            _, self.pos = read_int32(self.data, self.pos)
            return None
        elif rt == RT_MEMBER_PRIMITIVE_TYPED:
            self._parse_member_primitive_typed()
            return None
        else:
            return None

    def resolve_ref(self, val):
        """Resolve a reference to its actual value."""
        if isinstance(val, tuple) and len(val) == 2 and val[0] == "__ref__":
            obj = self.objects.get(val[1])
            return obj  # Return as-is (could be str, dict, or None)
        return val

    def fix_missed_text_alignments(self):
        """
        Post-parse binary scan for TextAlignment objects missed due to parser sync loss.

        When the main NRBF parser goes out of sync (e.g. a garbage record consumes
        bytes that belong to a real ClassWithMembersAndTypes), TextAlignment objects
        are never stored in self.objects, leaving all alignment refs unresolvable.

        This scanner finds them by searching for the class name bytes directly,
        parses the <index> field, and populates self.objects / self.class_defs so
        that alignment resolution works correctly.
        """
        data = self.data
        class_name = "UsbMonitorL.TextAlignment"
        # LPS-encoded class name: 1-byte length prefix + ASCII bytes
        pattern = bytes([len(class_name)]) + class_name.encode("ascii")

        ta_class_def = None
        ta_first_id = None

        # --- Pass 1: find ClassWithMembersAndTypes (0x05) records ---
        pos = 0
        while True:
            idx = data.find(pattern, pos)
            if idx == -1:
                break
            pos = idx + 1

            # Structure: [0x05][obj_id 4B][LPS name...]
            # idx = start of LPS length byte → idx-5 = record type, idx-4..idx-1 = obj_id
            if idx < 5 or data[idx - 5] != RT_CLASS_WITH_MEMBERS_AND_TYPES:
                continue

            obj_id = struct.unpack_from("<i", data, idx - 4)[0]
            if ta_first_id is None:
                ta_first_id = obj_id
            if obj_id in self.objects:
                if ta_class_def is None:
                    ta_class_def = self.class_defs.get(obj_id)
                continue

            # Manually parse the record starting after the class name LPS
            p = idx + len(pattern)
            try:
                member_count, p = read_int32(data, p)
                if member_count != 2:
                    continue
                member_names = []
                for _ in range(2):
                    name, p = read_lps(data, p)
                    member_names.append(name)
                mt0, p = read_byte(data, p)
                mt1, p = read_byte(data, p)
                ptype = None
                if mt1 == BT_PRIMITIVE:
                    ptype, p = read_byte(data, p)
                _, p = read_int32(data, p)  # library_id

                # member[0]: String (BinaryObjectString or MemberReference)
                display_name = None
                rt0 = data[p] if p < len(data) else -1
                if rt0 == RT_BINARY_OBJECT_STRING:
                    p += 1
                    str_id, p = read_int32(data, p)
                    display_name, p = read_lps(data, p)
                    self.objects[str_id] = display_name
                elif rt0 == RT_MEMBER_REFERENCE:
                    p += 1
                    ref_id, p = read_int32(data, p)
                    display_name = self.objects.get(ref_id)
                else:
                    continue

                # member[1]: Primitive Int32 = alignment index
                if p + 4 > len(data):
                    continue
                align_idx, p = read_int32(data, p)

                ai1 = ("primitive", ptype) if ptype is not None else (mt1,)
                class_def = {
                    "class_name": class_name,
                    "member_names": member_names,
                    "member_types": [mt0, mt1],
                    "additional_info": [(mt0,), ai1],
                }
                self.class_defs[obj_id] = class_def
                self.objects[obj_id] = {
                    "_class": class_name,
                    "_id": obj_id,
                    member_names[0]: display_name,
                    member_names[1]: align_idx,
                }
                if ta_class_def is None:
                    ta_class_def = class_def
            except (struct.error, IndexError, UnicodeDecodeError):
                continue

        if ta_first_id is None or ta_class_def is None:
            return

        # --- Pass 2: find ClassWithId (0x01) records referencing ta_first_id ---
        ref_bytes = struct.pack("<i", ta_first_id)
        pos = 0
        while True:
            idx = data.find(ref_bytes, pos)
            if idx == -1:
                break
            pos = idx + 1

            # Structure: [0x01][obj_id 4B][ref_id 4B][values...]
            # idx = start of ref_id bytes → idx-5 = record type, idx-4..idx-1 = obj_id
            if idx < 5 or data[idx - 5] != RT_CLASS_WITH_ID:
                continue

            obj_id = struct.unpack_from("<i", data, idx - 4)[0]
            if obj_id in self.objects:
                continue

            p = idx + 4  # member values start after ref_id
            try:
                display_name = None
                rt0 = data[p] if p < len(data) else -1
                if rt0 == RT_BINARY_OBJECT_STRING:
                    p += 1
                    str_id, p = read_int32(data, p)
                    display_name, p = read_lps(data, p)
                    self.objects[str_id] = display_name
                elif rt0 == RT_MEMBER_REFERENCE:
                    p += 1
                    ref_id, p = read_int32(data, p)
                    display_name = self.objects.get(ref_id)
                else:
                    continue

                if p + 4 > len(data):
                    continue
                align_idx, p = read_int32(data, p)

                names = ta_class_def["member_names"]
                self.class_defs[obj_id] = ta_class_def
                self.objects[obj_id] = {
                    "_class": class_name,
                    "_id": obj_id,
                    names[0]: display_name,
                    names[1]: align_idx,
                }
            except (struct.error, IndexError, UnicodeDecodeError):
                continue


# ---------------------------------------------------------------------------
# Theme Data Extraction
# ---------------------------------------------------------------------------

def extract_color_from_obj(obj: dict) -> Optional[tuple]:
    """Extract RGB from a System.Drawing.Color object.

    .NET Color has 3 modes (state field):
      state=0: empty/default (value=0)
      state=1: named/known color (use knownColor index)
      state=2: explicit ARGB (use value field)
    """
    if not isinstance(obj, dict):
        return None

    state = obj.get("state", 0)
    val = obj.get("value", 0)
    known = obj.get("knownColor", 0)

    if state == 2 and isinstance(val, int) and val != 0:
        # Explicit ARGB value
        a = (val >> 24) & 0xFF
        r = (val >> 16) & 0xFF
        g = (val >> 8) & 0xFF
        b = val & 0xFF
        return (r, g, b, a)

    if state == 1 and isinstance(known, int) and known > 0:
        # KnownColor index - lookup from .NET table
        rgb = KNOWN_COLORS.get(known)
        if rgb:
            return (rgb[0], rgb[1], rgb[2], 255)

    # Fallback: try value even if state is 0
    if isinstance(val, int) and val != 0:
        a = (val >> 24) & 0xFF
        r = (val >> 16) & 0xFF
        g = (val >> 8) & 0xFF
        b = val & 0xFF
        return (r, g, b, a)

    return None


# .NET System.Drawing.KnownColor table (index -> RGB)
# Source: https://learn.microsoft.com/en-us/dotnet/api/system.drawing.knowncolor
KNOWN_COLORS = {
    27: (240, 248, 255),    # AliceBlue
    28: (250, 235, 215),    # AntiqueWhite
    29: (0, 255, 255),      # Aqua
    30: (127, 255, 212),    # Aquamarine
    31: (240, 255, 255),    # Azure
    32: (245, 245, 220),    # Beige
    33: (255, 228, 196),    # Bisque
    34: (0, 0, 0),          # Black
    35: (0, 0, 0),          # Black (duplicate index used in some .NET versions)
    36: (255, 235, 205),    # BlanchedAlmond
    37: (0, 0, 255),        # Blue
    38: (138, 43, 226),     # BlueViolet
    39: (165, 42, 42),      # Brown
    40: (222, 184, 135),    # BurlyWood
    41: (95, 158, 160),     # CadetBlue
    42: (127, 255, 0),      # Chartreuse
    43: (210, 105, 30),     # Chocolate
    44: (255, 127, 80),     # Coral
    45: (100, 149, 237),    # CornflowerBlue
    46: (255, 248, 220),    # Cornsilk
    47: (220, 20, 60),      # Crimson
    48: (0, 255, 255),      # Cyan
    49: (0, 0, 139),        # DarkBlue
    50: (0, 139, 139),      # DarkCyan
    51: (184, 134, 11),     # DarkGoldenrod
    52: (169, 169, 169),    # DarkGray
    53: (0, 100, 0),        # DarkGreen
    54: (189, 183, 107),    # DarkKhaki
    55: (139, 0, 139),      # DarkMagenta
    56: (85, 107, 47),      # DarkOliveGreen
    57: (255, 140, 0),      # DarkOrange
    58: (153, 50, 204),     # DarkOrchid
    59: (139, 0, 0),        # DarkRed
    60: (233, 150, 122),    # DarkSalmon
    61: (143, 188, 139),    # DarkSeaGreen
    62: (72, 61, 139),      # DarkSlateBlue
    63: (47, 79, 79),       # DarkSlateGray
    64: (0, 206, 209),      # DarkTurquoise
    65: (148, 0, 211),      # DarkViolet
    66: (255, 20, 147),     # DeepPink
    67: (0, 191, 255),      # DeepSkyBlue
    68: (105, 105, 105),    # DimGray
    69: (30, 144, 255),     # DodgerBlue
    70: (178, 34, 34),      # Firebrick
    71: (255, 250, 240),    # FloralWhite
    72: (34, 139, 34),      # ForestGreen
    73: (255, 0, 255),      # Fuchsia
    74: (220, 220, 220),    # Gainsboro
    75: (248, 248, 255),    # GhostWhite
    76: (255, 215, 0),      # Gold
    77: (218, 165, 32),     # Goldenrod
    78: (128, 128, 128),    # Gray
    79: (0, 128, 0),        # Green
    80: (173, 255, 47),     # GreenYellow
    81: (240, 255, 240),    # Honeydew
    82: (255, 105, 180),    # HotPink
    83: (205, 92, 92),      # IndianRed
    84: (75, 0, 130),       # Indigo
    85: (255, 255, 240),    # Ivory
    86: (240, 230, 140),    # Khaki
    87: (230, 230, 250),    # Lavender
    88: (255, 240, 245),    # LavenderBlush
    89: (124, 252, 0),      # LawnGreen
    90: (255, 250, 205),    # LemonChiffon
    91: (173, 216, 230),    # LightBlue
    92: (240, 128, 128),    # LightCoral
    93: (224, 255, 255),    # LightCyan
    94: (250, 250, 210),    # LightGoldenrodYellow
    95: (144, 238, 144),    # LightGreen
    96: (211, 211, 211),    # LightGray
    97: (255, 182, 193),    # LightPink
    98: (255, 160, 122),    # LightSalmon
    99: (32, 178, 170),     # LightSeaGreen
    100: (135, 206, 250),   # LightSkyBlue
    101: (119, 136, 153),   # LightSlateGray
    102: (176, 196, 222),   # LightSteelBlue
    103: (255, 255, 224),   # LightYellow
    104: (0, 255, 0),       # Lime
    105: (50, 205, 50),     # LimeGreen
    106: (250, 240, 230),   # Linen
    107: (255, 0, 255),     # Magenta
    108: (128, 0, 0),       # Maroon
    109: (102, 205, 170),   # MediumAquamarine
    110: (0, 0, 205),       # MediumBlue
    111: (186, 85, 211),    # MediumOrchid
    112: (147, 112, 219),   # MediumPurple
    113: (60, 179, 113),    # MediumSeaGreen
    114: (123, 104, 238),   # MediumSlateBlue
    115: (0, 250, 154),     # MediumSpringGreen
    116: (72, 209, 204),    # MediumTurquoise
    117: (199, 21, 133),    # MediumVioletRed
    118: (25, 25, 112),     # MidnightBlue
    119: (245, 255, 250),   # MintCream
    120: (255, 228, 225),   # MistyRose
    121: (255, 228, 181),   # Moccasin
    122: (255, 222, 173),   # NavajoWhite
    123: (0, 0, 128),       # Navy
    124: (253, 245, 230),   # OldLace
    125: (128, 128, 0),     # Olive
    126: (107, 142, 35),    # OliveDrab
    127: (255, 165, 0),     # Orange
    128: (255, 69, 0),      # OrangeRed
    129: (218, 112, 214),   # Orchid
    130: (238, 232, 170),   # PaleGoldenrod
    131: (152, 251, 152),   # PaleGreen
    132: (175, 238, 238),   # PaleTurquoise
    133: (219, 112, 147),   # PaleVioletRed
    134: (255, 239, 213),   # PapayaWhip
    135: (255, 218, 185),   # PeachPuff
    136: (205, 133, 63),    # Peru
    137: (255, 192, 203),   # Pink
    138: (221, 160, 221),   # Plum
    139: (176, 224, 230),   # PowderBlue
    140: (128, 0, 128),     # Purple
    141: (255, 0, 0),       # Red
    142: (188, 143, 143),   # RosyBrown
    143: (65, 105, 225),    # RoyalBlue
    144: (139, 69, 19),     # SaddleBrown
    145: (250, 128, 114),   # Salmon
    146: (244, 164, 96),    # SandyBrown
    147: (46, 139, 87),     # SeaGreen
    148: (255, 245, 238),   # SeaShell
    149: (160, 82, 45),     # Sienna
    150: (192, 192, 192),   # Silver
    151: (135, 206, 235),   # SkyBlue
    152: (106, 90, 205),    # SlateBlue
    153: (112, 128, 144),   # SlateGray
    154: (255, 250, 250),   # Snow
    155: (0, 255, 127),     # SpringGreen
    156: (70, 130, 180),    # SteelBlue
    157: (210, 180, 140),   # Tan
    158: (0, 128, 128),     # Teal
    159: (216, 191, 216),   # Thistle
    160: (255, 99, 71),     # Tomato
    161: (64, 224, 208),    # Turquoise
    162: (238, 130, 238),   # Violet
    163: (245, 222, 179),   # Wheat
    164: (255, 255, 255),   # White
    165: (245, 245, 245),   # WhiteSmoke
    166: (255, 255, 0),     # Yellow
    167: (154, 205, 50),    # YellowGreen
}


def extract_theme_data(parser: NRBFParser) -> dict:
    """Extract structured theme data from parsed NRBF objects."""
    theme_data = {}

    # Find the Theme object (UsbMonitorL.Theme)
    theme_obj = None
    for obj in parser.objects.values():
        if isinstance(obj, dict) and "UsbMonitorL.Theme" in obj.get("_class", ""):
            theme_obj = obj
            break

    if not theme_obj:
        return theme_data

    # Extract basic theme properties
    def get_field(obj, field_suffix):
        """Get a field value by matching the end of the key name."""
        for k, v in obj.items():
            if k.endswith(field_suffix):
                val = parser.resolve_ref(v)
                return val
        return None

    theme_data["name"] = get_field(theme_obj, "<name>k__BackingField") or ""
    theme_data["width"] = get_field(theme_obj, "<width>k__BackingField") or 800
    theme_data["height"] = get_field(theme_obj, "<height>k__BackingField") or 480
    theme_data["is_landscape"] = get_field(theme_obj, "<isLanscape>k__BackingField")

    # Video path
    video_name = get_field(theme_obj, "<videoName>k__BackingField")
    video_target = get_field(theme_obj, "<videoTargetPath>k__BackingField")
    video_path = get_field(theme_obj, "<videoPath>k__BackingField")
    if video_name and isinstance(video_name, str) and video_name.strip():
        theme_data["video_name"] = video_name
    if video_target and isinstance(video_target, str):
        theme_data["video_target_path"] = video_target
    if video_path and isinstance(video_path, str):
        theme_data["video_path"] = video_path

    # Frame rate
    frame_rate = get_field(theme_obj, "<FrameRate>k__BackingField")
    if frame_rate and isinstance(frame_rate, (int, float)) and frame_rate > 0:
        theme_data["frame_rate"] = int(frame_rate)

    # Colors
    set_color = get_field(theme_obj, "<setColor>k__BackingField")
    front_color = get_field(theme_obj, "<frontColor>k__BackingField")
    back_color = get_field(theme_obj, "<backColor>k__BackingField")

    if isinstance(set_color, dict):
        rgb = extract_color_from_obj(set_color)
        if rgb:
            theme_data["set_color"] = rgb
    if isinstance(front_color, dict):
        rgb = extract_color_from_obj(front_color)
        if rgb:
            theme_data["front_color"] = rgb
    if isinstance(back_color, dict):
        rgb = extract_color_from_obj(back_color)
        if rgb:
            theme_data["back_color"] = rgb

    # Extract GraphItems
    graph_items = []
    font_configs = {}  # id -> parsed font config
    m_data_map = {}    # id -> parsed m_data

    # First pass: collect M_Data and FontConfig objects
    for oid, obj in parser.objects.items():
        if not isinstance(obj, dict):
            continue
        cls = obj.get("_class", "")
        if "M_Data" in cls:
            m_data_map[oid] = obj
        elif "FontConfig" in cls:
            font_configs[oid] = obj

    # Second pass: collect GraphItems
    for oid, obj in parser.objects.items():
        if not isinstance(obj, dict):
            continue
        cls = obj.get("_class", "")
        if "GraphItem" not in cls and "GraphImage" not in cls and "GraphArchBar" not in cls and "GraphStatuBar" not in cls:
            continue
        # Skip animation (video overlay)
        if "Animation" in cls:
            continue

        item = {}

        # Position and size — stored as INT32 in most themes, SINGLE/DOUBLE in some
        pos_x = get_field(obj, "<posX>k__BackingField")
        pos_y = get_field(obj, "<posY>k__BackingField")
        if isinstance(pos_x, (int, float)):
            item["x"] = int(round(pos_x))
        if isinstance(pos_y, (int, float)):
            item["y"] = int(round(pos_y))
        item_w = get_field(obj, "<width>k__BackingField")
        item_h = get_field(obj, "<height>k__BackingField")
        if isinstance(item_w, (int, float)) and item_w > 0:
            item["item_width"] = int(round(item_w))
        if isinstance(item_h, (int, float)) and item_h > 0:
            item["item_height"] = int(round(item_h))

        # Type info
        type_name = get_field(obj, "<TypeName>k__BackingField")
        if isinstance(type_name, tuple):
            type_name = parser.resolve_ref(type_name)
        if isinstance(type_name, str):
            item["type_name"] = type_name

        # Display name
        disp = obj.get("_DisplayName")
        if isinstance(disp, tuple):
            disp = parser.resolve_ref(disp)
        if isinstance(disp, str):
            item["display_name"] = disp

        # Hidden/enabled
        hide = get_field(obj, "<hide>k__BackingField")
        enabled = get_field(obj, "<enabled>k__BackingField")
        if isinstance(hide, bool):
            item["hidden"] = hide
        if isinstance(enabled, bool):
            item["enabled"] = enabled

        # Zoom rate (for GraphImage items)
        zoom = get_field(obj, "<zoom_rate>k__BackingField")
        if isinstance(zoom, float) and zoom > 0:
            item["zoom_rate"] = zoom

        # ArchBar (radial/curved bar) specific fields
        if "ArchBar" in cls:
            for field in ["<archWidth>k__BackingField", "<diameter>k__BackingField",
                          "<startPer>k__BackingField", "<totalAngel>k__BackingField"]:
                val = get_field(obj, field)
                if isinstance(val, (int, float)):
                    short = field.split("<")[1].split(">")[0]
                    item[short] = val
            # Colors: front (bar), back (empty track), gradient
            for color_suffix, dst_key in [
                ("<FrontColor>k__BackingField", "bar_color"),
                ("<BackColor>k__BackingField", "back_color"),
                ("<GradientColor>k__BackingField", "gradient_color"),
            ]:
                c_val = get_field(obj, color_suffix)
                if isinstance(c_val, dict):
                    rgb = extract_color_from_obj(c_val)
                    if rgb:
                        item[dst_key] = f"{rgb[0]}, {rgb[1]}, {rgb[2]}"
            # Booleans: round, revert, revert_value
            for src_suffix, dst_key in [
                ("<Round>k__BackingField", "round"),
                ("<isRound>k__BackingField", "round"),
                ("<Revert>k__BackingField", "revert"),
                ("<isRevert>k__BackingField", "revert"),
                ("<RevertValue>k__BackingField", "revert_value"),
                ("<revertValue>k__BackingField", "revert_value"),
            ]:
                if dst_key not in item:
                    v = get_field(obj, src_suffix)
                    if isinstance(v, bool):
                        item[dst_key] = v
            # Integer: block_angle (segmented arc mode)
            for src_suffix in ["<blockAngle>k__BackingField", "<BlockAngle>k__BackingField"]:
                v = get_field(obj, src_suffix)
                if isinstance(v, (int, float)) and v > 0:
                    item["block_angle"] = int(round(v))
                    break

        # StatusBar (progress bar) specific fields
        if "StatuBar" in cls:
            for field in ["<width>k__BackingField", "<height>k__BackingField",
                          "<radius>k__BackingField", "<direction>k__BackingField"]:
                val = get_field(obj, field)
                if isinstance(val, (int, float)):
                    short = field.split("<")[1].split(">")[0]
                    item[short] = val
            # Colors: front (bar), back (empty track), gradient
            for color_suffix, dst_key in [
                ("<FrontColor>k__BackingField", "bar_color"),
                ("<BackColor>k__BackingField", "back_color"),
                ("<GradientColor>k__BackingField", "gradient_color"),
            ]:
                c_val = get_field(obj, color_suffix)
                if isinstance(c_val, dict):
                    rgb = extract_color_from_obj(c_val)
                    if rgb:
                        item[dst_key] = f"{rgb[0]}, {rgb[1]}, {rgb[2]}"
            # Integer fields: block_width, corner_radius, border_width
            for src_suffix, dst_key in [
                ("<blockWidth>k__BackingField", "block_width"),
                ("<BlockWidth>k__BackingField", "block_width"),
                ("<cornerRadius>k__BackingField", "corner_radius"),
                ("<CornerRadius>k__BackingField", "corner_radius"),
                ("<borderWidth>k__BackingField", "border_width"),
                ("<BorderWidth>k__BackingField", "border_width"),
            ]:
                if dst_key not in item:
                    v = get_field(obj, src_suffix)
                    if isinstance(v, (int, float)) and v > 0:
                        item[dst_key] = int(round(v))
            # Boolean: revert_value
            for src_suffix in ["<RevertValue>k__BackingField", "<revertValue>k__BackingField"]:
                if "revert_value" not in item:
                    v = get_field(obj, src_suffix)
                    if isinstance(v, bool):
                        item["revert_value"] = v

        # M_Data reference
        m_data_val = get_field(obj, "<m_data>k__BackingField")
        m_obj = None
        if isinstance(m_data_val, dict):
            m_obj = m_data_val
        elif isinstance(m_data_val, tuple) and m_data_val[0] == "__ref__":
            m_obj = parser.objects.get(m_data_val[1])
        if isinstance(m_obj, dict):
            data_name = get_field(m_obj, "<DataName>k__BackingField")
            if isinstance(data_name, tuple):
                data_name = parser.resolve_ref(data_name)
            if isinstance(data_name, str):
                item["data_name"] = data_name
            show_unit = get_field(m_obj, "<ShowUnit>k__BackingField")
            if isinstance(show_unit, bool):
                item["show_unit"] = show_unit
            sub_name = get_field(m_obj, "<SubName>k__BackingField")
            if isinstance(sub_name, str) and sub_name:
                item["sub_name"] = sub_name
            # For static text, get the ValueWithUnit (= the static text content)
            vwu = get_field(m_obj, "<ValueWithUnit>k__BackingField")
            if isinstance(vwu, tuple):
                vwu = parser.resolve_ref(vwu)
            if isinstance(vwu, str):
                if data_name == "StaticText":
                    item["static_text"] = vwu
                elif vwu:
                    item["placeholder"] = vwu

        # FontConfig reference
        fc_val = get_field(obj, "<fontConfig>k__BackingField")
        fc_obj = None
        if isinstance(fc_val, dict):
            fc_obj = fc_val
        elif isinstance(fc_val, tuple) and fc_val[0] == "__ref__":
            fc_obj = parser.objects.get(fc_val[1])
        if isinstance(fc_obj, dict):
            font_name = get_field(fc_obj, "<name>k__BackingField")
            if isinstance(font_name, tuple):
                font_name = parser.resolve_ref(font_name)
            font_size = get_field(fc_obj, "<size>k__BackingField")
            is_bold = get_field(fc_obj, "<isBold>k__BackingField")

            fc = {}
            if isinstance(font_name, str):
                fc["font"] = font_name
            if isinstance(font_size, int):
                fc["size"] = font_size
            if isinstance(is_bold, bool):
                fc["bold"] = is_bold

            # Alignment from FontConfig — always emit so the renderer never guesses
            align_val = get_field(fc_obj, "<alignment>k__BackingField")
            if isinstance(align_val, dict):
                align_idx = align_val.get("<index>k__BackingField")
                if align_idx == 1:
                    fc["align"] = "center"
                elif align_idx == 2:
                    fc["align"] = "right"
                else:
                    fc["align"] = "left"

            # Color from FontConfig
            color_val = get_field(fc_obj, "<color>k__BackingField")
            c_obj = None
            if isinstance(color_val, dict):
                c_obj = color_val
            elif isinstance(color_val, tuple) and color_val[0] == "__ref__":
                c_obj = parser.objects.get(color_val[1])
            if isinstance(c_obj, dict):
                rgb = extract_color_from_obj(c_obj)
                if rgb:
                    fc["color"] = rgb[:3]

            if fc:
                item["font_config"] = fc

        if item.get("data_name") or item.get("type_name"):
            graph_items.append(item)

    theme_data["graph_items"] = graph_items

    # Extract StatusBar items (progress bars)
    status_bars = []
    for oid, obj in parser.objects.items():
        if not isinstance(obj, dict):
            continue
        cls = obj.get("_class", "")
        if "GraphStatuBar" not in cls and "StatusBar" not in cls:
            continue
        bar = {}
        for field in ["<direction>k__BackingField", "<width>k__BackingField",
                      "<height>k__BackingField", "<radius>k__BackingField",
                      "<lineWidth>k__BackingField"]:
            val = get_field(obj, field)
            if isinstance(val, (int, float)):
                short_name = field.split("<")[1].split(">")[0]
                bar[short_name] = val
        # Front/back colors
        fc_ref = get_field(obj, "<FrontColor>k__BackingField")
        c_obj = None
        if isinstance(fc_ref, dict):
            c_obj = fc_ref
        elif isinstance(fc_ref, tuple) and fc_ref[0] == "__ref__":
            c_obj = parser.objects.get(fc_ref[1])
        if isinstance(c_obj, dict):
            rgb = extract_color_from_obj(c_obj)
            if rgb:
                bar["front_color"] = rgb[:3]
        if bar:
            status_bars.append(bar)

    if status_bars:
        theme_data["status_bars"] = status_bars

    # Extract ArchBar items (radial gauges)
    arch_bars = []
    for oid, obj in parser.objects.items():
        if not isinstance(obj, dict):
            continue
        cls = obj.get("_class", "")
        if "GraphArchBar" not in cls:
            continue
        bar = {}
        for field in ["<archWidth>k__BackingField", "<diameter>k__BackingField",
                      "<startPer>k__BackingField", "<totalAngel>k__BackingField",
                      "<height>k__BackingField"]:
            val = get_field(obj, field)
            if isinstance(val, (int, float)):
                short_name = field.split("<")[1].split(">")[0]
                bar[short_name] = val
        fc_ref = get_field(obj, "<FrontColor>k__BackingField")
        c_obj = None
        if isinstance(fc_ref, dict):
            c_obj = fc_ref
        elif isinstance(fc_ref, tuple) and fc_ref[0] == "__ref__":
            c_obj = parser.objects.get(fc_ref[1])
        if isinstance(c_obj, dict):
            rgb = extract_color_from_obj(c_obj)
            if rgb:
                bar["front_color"] = rgb[:3]
        if bar:
            arch_bars.append(bar)

    if arch_bars:
        theme_data["arch_bars"] = arch_bars

    return theme_data


# ---------------------------------------------------------------------------
# Binary Image Extraction (by signature scanning)
# ---------------------------------------------------------------------------

def find_png_end(data: bytes, start: int) -> int:
    iend = b"IEND\xae\x42\x60\x82"
    idx = data.find(iend, start)
    if idx == -1:
        return min(start + 10 * 1024 * 1024, len(data))
    return idx + 8

def find_jpeg_end(data: bytes, start: int) -> int:
    idx = data.find(b"\xff\xd9", start + 2)
    if idx == -1:
        return len(data)
    return idx + 2

def extract_images(data: bytes) -> list[dict]:
    """Extract embedded images by scanning for PNG/JPEG signatures."""
    images = []
    found_ranges = []

    # PNG
    sig = b"\x89PNG\r\n\x1a\n"
    pos = 0
    count = 0
    while True:
        idx = data.find(sig, pos)
        if idx == -1:
            break
        end = find_png_end(data, idx)
        images.append({
            "type": "png",
            "offset": idx,
            "size": end - idx,
            "data": data[idx:end],
            "filename": f"image_{count}.png",
        })
        found_ranges.append((idx, end))
        count += 1
        pos = end  # skip past this image

    # JPEG
    sig = b"\xff\xd8\xff"
    pos = 0
    jcount = 0
    while True:
        idx = data.find(sig, pos)
        if idx == -1:
            break
        if any(s <= idx < e for s, e in found_ranges):
            pos = idx + 1
            continue
        end = find_jpeg_end(data, idx)
        images.append({
            "type": "jpeg",
            "offset": idx,
            "size": end - idx,
            "data": data[idx:end],
            "filename": f"image_{jcount}.jpg",
        })
        found_ranges.append((idx, end))
        jcount += 1
        pos = end

    return images


# ---------------------------------------------------------------------------
# Main Extraction Logic
# ---------------------------------------------------------------------------

def extract_turtheme(filepath: str, output_dir: str) -> dict:
    """Extract all content from a .turtheme file."""
    filepath = Path(filepath)
    data = filepath.read_bytes()
    print(f"Processing: {filepath.name} ({len(data):,} bytes)")

    # 1. Parse NRBF structure
    print("  Parsing NRBF structure...")
    parser = NRBFParser(data)
    parser.parse()
    parser.fix_missed_text_alignments()
    print(f"    Parsed {len(parser.objects)} objects")

    # 2. Extract theme data from parsed objects
    print("  Extracting theme data...")
    theme_data = extract_theme_data(parser)

    # 3. Extract embedded images (skip byte-for-byte duplicates, keep original indices)
    print("  Extracting images...")
    images = extract_images(data)
    assets_dir = Path(output_dir) / "assets"
    assets_dir.mkdir(parents=True, exist_ok=True)
    asset_info = []
    seen_hashes: set[str] = set()
    for img in images:
        digest = hashlib.md5(img["data"]).hexdigest()
        if digest in seen_hashes:
            print(f"    skipped duplicate {img['filename']} ({img['size']:,} bytes)")
            continue
        seen_hashes.add(digest)
        # Keep original filename so the converter's index-based logic stays valid
        out_path = assets_dir / img["filename"]
        out_path.write_bytes(img["data"])
        asset_info.append({
            "type": img["type"],
            "filename": img["filename"],
            "size": img["size"],
        })
        print(f"    {img['type']}: {img['filename']} ({img['size']:,} bytes)")
    theme_data["assets"] = asset_info

    # 4. Write YAML output
    yaml_path = Path(output_dir) / "theme.yaml"
    # Clean up for YAML output
    yaml_out = {}
    yaml_out["name"] = theme_data.get("name", filepath.stem)
    yaml_out["width"] = theme_data.get("width", 800)
    yaml_out["height"] = theme_data.get("height", 480)
    yaml_out["is_landscape"] = theme_data.get("is_landscape")
    if theme_data.get("video_name"):
        yaml_out["video_name"] = theme_data["video_name"]
    if theme_data.get("video_target_path"):
        yaml_out["video_target_path"] = theme_data["video_target_path"]
    if theme_data.get("frame_rate"):
        yaml_out["frame_rate"] = theme_data["frame_rate"]
    if theme_data.get("set_color"):
        r, g, b, a = theme_data["set_color"]
        yaml_out["set_color"] = f"{r}, {g}, {b}"
    if theme_data.get("front_color"):
        r, g, b, a = theme_data["front_color"]
        yaml_out["front_color"] = f"{r}, {g}, {b}"
    yaml_out["assets"] = asset_info

    # GraphItems
    items_out = []
    for item in theme_data.get("graph_items", []):
        entry = {}
        if item.get("data_name"):
            entry["data_name"] = item["data_name"]
        if item.get("type_name"):
            entry["type_name"] = item["type_name"]
        if item.get("display_name"):
            entry["display_name"] = item["display_name"]
        if "x" in item:
            entry["x"] = item["x"]
        if "y" in item:
            entry["y"] = item["y"]
        if "item_width" in item:
            entry["item_width"] = item["item_width"]
        if "item_height" in item:
            entry["item_height"] = item["item_height"]
        if item.get("hidden"):
            entry["hidden"] = item["hidden"]
        if item.get("enabled") is False:
            entry["enabled"] = False
        if item.get("show_unit"):
            entry["show_unit"] = item["show_unit"]
        if item.get("sub_name"):
            entry["sub_name"] = item["sub_name"]
        if item.get("static_text"):
            entry["static_text"] = item["static_text"]
        if item.get("placeholder"):
            entry["placeholder"] = item["placeholder"]
        if item.get("zoom_rate"):
            entry["zoom_rate"] = item["zoom_rate"]
        if item.get("archWidth"):
            entry["archWidth"] = item["archWidth"]
        if item.get("diameter"):
            entry["diameter"] = item["diameter"]
        if item.get("startPer"):
            entry["startPer"] = item["startPer"]
        if item.get("totalAngel"):
            entry["totalAngel"] = item["totalAngel"]
        if item.get("bar_color"):
            entry["bar_color"] = item["bar_color"]
        if item.get("width") and item.get("type_name") == "StatuBar":
            entry["width"] = item["width"]
        if item.get("height") and item.get("type_name") == "StatuBar":
            entry["height"] = item["height"]
        if item.get("font_config"):
            fc = item["font_config"]
            entry["font"] = fc.get("font", "")
            entry["font_size"] = fc.get("size", 12)
            if fc.get("bold"):
                entry["font_bold"] = True
            if fc.get("color"):
                r, g, b = fc["color"]
                entry["font_color"] = f"{r}, {g}, {b}"
            if fc.get("align"):
                entry["align"] = fc["align"]
        items_out.append(entry)
    yaml_out["graph_items"] = items_out

    # Status bars
    if theme_data.get("status_bars"):
        # Convert tuples to strings in status_bars
        bars_out = []
        for bar in theme_data["status_bars"]:
            bar_out = {}
            for k, v in bar.items():
                if isinstance(v, tuple):
                    bar_out[k] = f"{v[0]}, {v[1]}, {v[2]}"
                else:
                    bar_out[k] = v
            bars_out.append(bar_out)
        yaml_out["status_bars"] = bars_out
    if theme_data.get("arch_bars"):
        bars_out = []
        for bar in theme_data["arch_bars"]:
            bar_out = {}
            for k, v in bar.items():
                if isinstance(v, tuple):
                    bar_out[k] = f"{v[0]}, {v[1]}, {v[2]}"
                else:
                    bar_out[k] = v
            bars_out.append(bar_out)
        yaml_out["arch_bars"] = bars_out

    with open(yaml_path, "w", encoding="utf-8") as f:
        yaml.dump(yaml_out, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
    print(f"  Written: {yaml_path}")

    n_items = len(items_out)
    n_data = len([i for i in items_out if i.get("data_name") and i["data_name"] != "StaticText"])
    n_text = len([i for i in items_out if i.get("data_name") == "StaticText" or i.get("type_name") == "Text"])
    print(f"  Summary: {n_data} sensors, {n_text} static texts, {len(images)} images")
    return theme_data


def process_directory(directory: str, output_base: str, all_files: bool = False):
    """Process turtheme files in a directory."""
    dir_path = Path(directory)
    if all_files:
        files = sorted(dir_path.glob("*.turtheme"))
    else:
        files = list(dir_path.glob("*.turtheme"))
    if not files:
        print(f"No .turtheme files found in {directory}")
        return
    print(f"Found {len(files)} .turtheme files\n")
    for f in files:
        theme_name = f.stem
        output_dir = Path(output_base) / theme_name
        output_dir.mkdir(parents=True, exist_ok=True)
        try:
            extract_turtheme(str(f), str(output_dir))
        except Exception as e:
            print(f"  ERROR: {e}")
        print()


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Extract content from .turtheme files")
    parser.add_argument("input", help="File or directory to process")
    parser.add_argument("--output-dir", "-o", default="extracted_all", help="Output base directory")
    parser.add_argument("--all", "-a", action="store_true", help="Process all .turtheme files")
    args = parser.parse_args()

    input_path = Path(args.input)
    if input_path.is_file():
        output_dir = Path(args.output_dir) / input_path.stem
        output_dir.mkdir(parents=True, exist_ok=True)
        extract_turtheme(str(input_path), str(output_dir))
    elif input_path.is_dir():
        process_directory(str(input_path), args.output_dir, all_files=args.all)
    else:
        print(f"Error: {args.input} not found")
        sys.exit(1)


if __name__ == "__main__":
    main()
