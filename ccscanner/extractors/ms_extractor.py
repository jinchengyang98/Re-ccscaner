import logging


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_xml, remove_rstrip

logging.basicConfig()
logger = logging.getLogger(__name__)


class MsExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'ms'


    def run_extractor(self):
        self.parse_ms()


    def parse_ms(self):
        content = read_xml(self.target)
        deps = content.find_all('AdditionalDependencies')
        for dep in deps:
            if len(dep.contents) == 0:
                continue
            dep_names = remove_rstrip(dep.contents[0], ';%(AdditionalDependencies)').split(';')
            for dep_name in dep_names:
                dep_name = dep_name.split('.')[0]
                dep = Dependency(dep_name, None)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)
            
        
        