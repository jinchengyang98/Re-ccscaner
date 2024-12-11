import re

RX_VERSION = re.compile("\\d+(\\.\\d+){1,6}([._-]?(snapshot|release|final|alpha|beta|rc$|[a-zA-Z]{1,3}[_-]?\\d{1,8}|[a-z]\\b|\\d{1,8}\\b))?", re.IGNORECASE)
RX_SINGLE_VERSION = re.compile("\\d+(\\.\\d+){0,6}([._-]?(snapshot|release|final|alpha|beta|rc$|[a-zA-Z]{1,3}[_-]?\\d{1,8}))?")

# pattern = '(roll|update|upgrade|bump)\s((\w+\s+){1,10})to\s((\w+\s+){0,3})2[0-9]+(\.[0-9]+)+'

def parse_version(text, first_match_only):
    vers = RX_VERSION.finditer(text)
    version = None
    ver = next(vers, None)
    if ver:
        version = ver.group()
    if first_match_only and next(vers, None):
        return None
    if version is None:
        vers = RX_SINGLE_VERSION.finditer(text)
        ver = next(vers, None)
        if ver:
            version = ver.group()
        else:
            return None
        if next(vers, None):
            return None
    return version
