import os
import re
import logging
import json


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_js, read_json5
logging.basicConfig()
logger = logging.getLogger(__name__)


class DdsExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'dds'

    def run_extractor(self):
        self.parse_dds()


    def parse_dds(self):
        contents = read_json5(self.target)
        if 'depends' not in contents:
            return
        for item in contents['depends']:
            dep_name = item
            version = contents['depends'][item]
            ## get unified name and parse version operator
            dep = Dependency(dep_name, version)
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)