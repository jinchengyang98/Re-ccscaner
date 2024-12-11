import os
import re
import logging
import json


from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_lines, read_js, read_txt, get_func_body
from ccscanner.config import BUCKAROO_REPOS_PARENT
from ccscanner.dataset.library_dataset.github_data import get_owner_name_from_github_url



logging.basicConfig()
logger = logging.getLogger(__name__)


class BuckExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.target = target
        self.type = 'buck'
        self.buckaroo_parents = read_js(BUCKAROO_REPOS_PARENT)
    def run_extractor(self):
        self.parse_buck()

# deps = [“:VeryCoolAppAssets”]

    def parse_buck(self):
        contents = read_txt(self.target)
        pattern = 'buckaroo_deps_from_package\s*\('
        funcs = get_func_body(pattern, contents)
        arg_pattern = '\((.*)\)'
        for func in funcs:
            url = re.search(arg_pattern, func).group(1).split(',')[0].strip('\"\'')
            owner, dep_name = get_owner_name_from_github_url(url)
            if owner == 'buckaroo-pm' and dep_name not in self.buckaroo_parents:
                continue
            if dep_name in self.buckaroo_parents:
                owner, dep_name = self.buckaroo_parents[dep_name].split('/')[:2]
            dep = Dependency(dep_name, None)
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)