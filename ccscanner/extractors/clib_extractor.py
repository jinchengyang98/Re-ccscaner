import os
import logging


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency

from ccscanner.utils.utils import read_js
logging.basicConfig()
logger = logging.getLogger(__name__)

## Explanation of clib.json / package.json
## https://github.com/clibs/clib/wiki/Explanation-of-clib.json
KEYS = ['name', 'version', 'src', 'dependencies', 'development', 'repo', 'description', 'keywords', 'license', 'makefile', 'install', 'uninstall']

class ClibExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'clib'


    def run_extractor(self):
        self.parse_clib()


    def parse_clib(self):
        content = read_js(self.target)
        if content is None:
            return
        if any(i not in KEYS for i in content) or 'name' not in content:
            logger.info("not clibs: " + self.target)
            return
        dep_name = content['name']
        version = content['version'] if 'version' in content else None
        dep = Dependency(dep_name, version)
        dep.add_evidence(self.type, self.target, 'High')
        self.add_dependency(dep)
        if 'dependencies' in content:
            for dep in content['dependencies']:
                dep_name = dep.split('/')[-1]
                version = content['dependencies'][dep]
                dep = Dependency(dep_name, version)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)
        if 'development' in content:
            for dep in content['development']:
                dep_name = dep.split('/')[-1]
                version = content['development'][dep]
                dep = Dependency(dep_name, version)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)
        
