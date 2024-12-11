import os
import logging
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_lines, read_txt, remove_lstrip

logging.basicConfig()
logger = logging.getLogger(__name__)

    
class MakeExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.type = 'make'
        self.target = target

    def run_extractor(self):
        self.process_make()

    def process_make(self):
        lines = read_lines(self.target)
        for line in lines:
            line = line.strip()
            if line.startswith('LDLIBS'):
                if '=' not in line:
                    continue
                libs = line.split('=')[1].split(' ')
                for lib in libs:
                    if lib.startswith('-l'):
                        dep_name = remove_lstrip(lib, '-l')
                        dep = Dependency(dep_name, None)
                        dep.add_evidence(self.type, self.target+':'+line, 'High')
                        self.add_dependency(dep)
