import os
import re
import logging
import json


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_lines, read_txt

logging.basicConfig()
logger = logging.getLogger(__name__)


class Build2Extractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'build2'

    def run_extractor(self):
        self.parse_build2()

    def parse_build2(self):
        contents = read_txt(self.target)
        if 'build2' not in contents:
            return
        lines = read_lines(self.target)
        for line in lines:
            line = line.strip()
            if line.startswith('depends:'):
                items = line.split(':', 1)[-1]
                items = items.strip('* ').split(' ', 1)
                dep_name = items[0]
                ## TODO: parse version string
                version = items[1]
                dep = Dependency(dep_name, version)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)