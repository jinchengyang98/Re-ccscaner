import logging
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_lines

logging.basicConfig()
logger = logging.getLogger(__name__)

    
class PkgExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.type = 'pkgconfig'
        self.target = target

    def run_extractor(self):
        self.process_pkg()

    def process_pkg(self):
        operators = ['>=', '<=', '=', '<', '>']
        lines = read_lines(self.target)
        dep_name = version = None
        for line in lines:
            if line.startswith('Name:'):
                dep_name = line.split(':', 1)[-1].strip()
            if line.startswith('Version:'):
                version = line.split(':', 1)[-1].strip()
            if line.startswith('Requires:'):
                if ',' in line:
                    requires = line.split(':', 1)[-1].strip().split(',')
                else:
                    requires = []
                    if ' ' in line:
                        tokens = line.split(':', 1)[-1].strip().split(' ')
                        state_operator = 0
                        operator = ''
                        name = ''
                        for token in tokens:
                            if token == ' ':
                                continue
                            if state_operator == 1:
                                item = name+operator+token
                                requires.append(item)
                                state_operator = 0
                                name = ''
                                operator = ''
                                continue
                            if token in operators:
                                state_operator = 1
                                operator = token
                                continue
                            if name:
                                requires.append(name)
                                name = token
                            else:
                                name = token
                        if name:
                            requires.append(name)
                            
                for require in requires:
                    req_name = req_version = version_operator = None
                    for operator in operators:
                        if operator in require:
                            req_name, req_version = require.split(operator)
                            version_operator = operator
                            break
                    if req_name is None:
                        req_name = require
                    dep = Dependency(req_name, req_version, version_operator)
                    dep.add_evidence(self.type, self.target, 'High')
                    self.add_dependency(dep)
        if dep_name is not None:
            dep = Dependency(dep_name, version, '=')
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)