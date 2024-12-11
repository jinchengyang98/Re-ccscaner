import logging
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_js

logging.basicConfig()
logger = logging.getLogger(__name__)

    
class VcpkgExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.type = 'vcpkg'
        self.target = target

    def run_extractor(self):
        self.process_vcpkg()

    def process_vcpkg(self):
        content = read_js(self.target)
        # TODO add dependencies in feature field
        if 'dependencies' in content:
            for dependency in content['dependencies']:
                if isinstance(dependency, str):
                    dep = Dependency(dependency, None)
                    dep.add_evidence(self.type, self.target, 'High')
                    self.add_dependency(dep)
                elif isinstance(dependency, dict):
                    if 'name' not in dependency:
                        continue
                    if 'version' in dependency:
                        version = dependency['version']
                    else:
                        version = None
                    dep = Dependency(dependency['name'], version)
                    dep.add_evidence(self.type, self.target, 'High')
                    self.add_dependency(dep)