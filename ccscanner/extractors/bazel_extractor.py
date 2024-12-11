import os
import re
import logging
import json


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_txt, get_func_body

logging.basicConfig()
logger = logging.getLogger(__name__)


class BazelExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'bazel'

    def run_extractor(self):
        self.parse_bazel()

    def parse_bazel(self):
        ## TODO: http_archive, cc_import
        contents = read_txt(self.target)
        pattern1 = 'cc_library\s*\('
        pattern2 = 'cc_binary\s*\('
        # args_pattern = 'deps=\[(.*)\]'
        args_pattern = 'deps=(\'.*?\'|\[.*?\])'
        dep_pattern = "\"(.*?)\""
        funcs1 = get_func_body(pattern1, contents)
        funcs2 = get_func_body(pattern2, contents)
        funcs = funcs1 + funcs2
        for func in funcs:
            func = func.replace('\n', '').replace(' ', '')
            if 'deps=' not in func:
                continue
            deps = re.search(args_pattern, func)
            if deps is None:
                continue
            deps = deps.group(1).strip('\'\"')
            if not deps.startswith('['):
                if ':' in deps:
                    deps = deps.split(':')[-1]
                dep = Dependency(deps, None)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)
                continue
            # deps = json.loads(deps.replace(',]', ']'))
            deps = deps.strip('[],').split(',')
            # deps = re.finditer(dep_pattern, deps)
            for context in deps:
                # dep_name = dep.group(1)
                dep_name = context.strip('\'\"')
                if ':' in dep_name:
                    dep_name = dep_name.split(':')[-1]
                dep = Dependency(dep_name, None)
                dep.add_evidence(self.type, self.target+':'+context, 'High')
                self.add_dependency(dep)
