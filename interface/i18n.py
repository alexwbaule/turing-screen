"""
i18n bootstrap — call setup() once before creating any GTK widgets.
After setup(), every module that does `from i18n import _` gets the
active translation function.
"""
import gettext as _gettext
import locale
import os

_DOMAIN = "turing-screen"
_LOCALEDIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "locale")


def _(s: str) -> str:
    return s


def setup(lang: str | None = None) -> None:
    """Activate translations. Call once at startup before building UI."""
    global _

    if lang is None:
        try:
            lang = locale.getdefaultlocale()[0] or "en"
        except Exception:
            lang = "en"

    candidates = [lang]
    if "_" in lang:
        candidates.append(lang.split("_")[0])
    candidates.append("en")

    try:
        t = _gettext.translation(
            _DOMAIN, localedir=_LOCALEDIR, languages=candidates
        )
        _ = t.gettext
    except FileNotFoundError:
        _ = lambda s: s  # noqa: E731
