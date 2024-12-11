import os
import re
import logging


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_txt, get_func_body

logging.basicConfig()
logger = logging.getLogger(__name__)


class XmakeExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'xmake'


    def run_extractor(self):
        self.parse_xmake()


    def parse_xmake(self):
        contents = read_txt(self.target)
        ## TODO: add_deps
        pattern = 'add_requires\s*\('
        funcs = get_func_body(pattern, contents)
        arg_pattern = '\((.*)\)'
        dic_pattern = '\{.*\}'
        for func in funcs:
            args = re.sub(dic_pattern, '', func)
            args = re.search(arg_pattern, args).group(1).split(',')
            for arg in args:
                arg = arg.strip(' ')
                if not arg.startswith('"'):
                    continue
                arg = arg.strip('"')
                if ' ' in arg:
                    dep_name, version = arg.split(' ')[:2]
                else:
                    version = None
                    dep_name = arg
                if '::' in dep_name:
                    dep_name = dep_name.split('::')[-1]
                dep = Dependency(dep_name, version)
                dep.add_evidence(self.type, arg, 'High')
                self.add_dependency(dep)
