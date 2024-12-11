import os
import logging
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_js, read_lines, read_txt

logging.basicConfig()
logger = logging.getLogger(__name__)

    
class ConanExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.type = 'conan'
        self.target = target

    def run_extractor(self):
        self.process_conan()

    def process_conan(self):
        basename = os.path.basename(self.target)
        if basename.lower() in ['conanfile.txt', 'conaninfo.txt']:
            self.process_conanfiletxt()
        elif basename.lower() == 'conanfile.py':
            self.process_conanfilepy()
        

    def process_conanfiletxt(self):
        lines = read_lines(self.target)
        flag = 0
        key_lines = ['[requires]', '[build_requires]', '[full_requires]']
        for line in lines:
            line = line.strip()
            if line in key_lines:
                flag = 1
                continue
            if flag != 1 or line is None:
                continue
            if line.startswith('[') and line not in key_lines:
                flag = 0
            dep_name, version = ConanExtractor.parse_conan_package(line)
            dep = Dependency(dep_name, version)
            dep.add_evidence(self.type, self.target+':'+line, 'High')
            self.add_dependency(dep)


    def process_conanfilepy(self):
        lines = read_lines(self.target)
        for line in lines:
            line = line.strip().replace(' ', '')
            if line.startswith(('requires=', 'build_requires =')):
                requires = line.split('=', 1)[-1]
                for require in requires.split(','):
                    req_name, req_version = ConanExtractor.parse_conan_package(require)
                    dep = Dependency(req_name, req_version)
                    dep.add_evidence(self.type, self.target+':'+require, 'High')
                    self.add_dependency(dep)
            if line.startswith(('self.requires(', 'self.build_requires(')):
                package = line.replace('self.requires(', '').strip('()')
                package = package.replace('self.build_requires(', '').strip('()')
                req_name, req_version = ConanExtractor.parse_conan_package(package)
                dep = Dependency(req_name, req_version)
                dep.add_evidence(self.type, self.target+':'+package, 'High')
                self.add_dependency(dep)

    @staticmethod
    def parse_conan_package(package):
        package = package.strip().strip('"')
        if '/' in package:
            package_name, version = package.split('/', 1)[:2]
            if '@' in version:
                version = version.split('@')[0]
        else:
            package_name = package
            version = None
        return package_name, version