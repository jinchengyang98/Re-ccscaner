import os
import re
import logging


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_txt, get_func_body
from ccscanner.utils.version import parse_version_str

logging.basicConfig()
logger = logging.getLogger(__name__)


class MesonExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'meson'

    def run_extractor(self):
        self.parse_meson()

    def parse_meson(self):
        ## TODO: declare_dependency
        contents = read_txt(self.target)
        pattern = 'dependency\s*\('
        args_pattern = 'dependency\((.*)\)'
        version_pattern = 'version:(\'.*?\'|\[.*?\])'
        funcs = get_func_body(pattern, contents)
        for func in funcs:
            if 'declare_'+func in contents:
                continue
            func = func.replace('\n', '').replace(' ', '')
            args = re.search(args_pattern, func).group(1)
            dep_name = args.split(',')[0].strip('\'\"')
            version = op = None
            if 'version:' in args:
                ## TODO: https://github.com/swaywm/sway/blob/master/meson.build
                ## replace variables with their values
                version = re.search(version_pattern, args)
                if version is not None:
                    version = version.group(1).strip('\'\"')
                    if not version.startswith('['):
                        _, version, op = parse_version_str(version)
            dep = Dependency(dep_name, version, op)
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)
