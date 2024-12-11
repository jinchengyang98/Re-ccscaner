import json
import re
from ccscanner.utils.utils import remove_lstrip, remove_rstrip

VERSION_SUFFIX_PATTERN = '[._-]?\d+(\.\d+){1,6}([._-]?(snapshot|release|final|alpha|beta|rc$|[a-zA-Z]{1,3}[_-]?\d{1,8}))?$'

class Dependency(object):
    def __init__(self, dep_name, version, operator = None) -> None:
        super().__init__()
        self.depname = dep_name
        self.version = version
        self.version_op = operator
        self.add_unified_name()

    def add_evidence(self, extractor_type, context, confidence):
        self.extractor_type = extractor_type
        self.context = context
        self.confidence = confidence
    
    def add_unified_name(self):
        name = remove_lstrip(self.depname.lower(), 'lib')
        while(name.endswith(('_find', '_major', '_minor', '_min', '_patchlevel', '_patch', '_debug'))):
            name = '_'.join(name.split('_')[:-1])
        for suffix in ['-dev', '_dev']:
            name = remove_rstrip(name, suffix)
        version = re.search(VERSION_SUFFIX_PATTERN, name)
        if version is not None:
            name = name.replace(version.group(0), '')
            version = version.group(0).strip('._-')
            if self.version is None:
                self.version = version
        self.unified_name = name

        
    def get_lib_version(self):
        return self.library, self.version

    def to_dict(self):
        return json.loads(json.dumps(self, default=lambda o: o.__dict__))