# import os
# import re
# import logging
# import json


# from ccscanner.extractors.extractor import Extractor
# from ccscanner.extractors.dependency import Dependency
# from ccscanner.utils.utils import read_lines, read_js
# from ccscanner.config import BUCKAROO_REPOS_PARENT
# from ccscanner.dataset.library_dataset.github_data import get_owner_name_from_github_url



# logging.basicConfig()
# logger = logging.getLogger(__name__)


# class BuckarooExtractor(Extractor):
#     def __init__(self, target) -> None:
#         super().__init__()
#         self.target = target
#         self.type = 'buckaroo'
#         self.buckaroo_parents = read_js(BUCKAROO_REPOS_PARENT)

#     def run_extractor(self):
#         self.parse_buckaroo()

#     def parse_buckaroo(self):
#         lines = read_lines(self.target)
#         flag = False
#         for line in lines:
#             line = line.replace(' ', '').strip()
#             if line == '[[dependency]]':
#                 flag = True
#                 dep_name = None
#                 version = None
#                 continue
#             if line.startswith('package=') and flag:
#                 url = line.split('package=')[-1].strip('\"\'')
#                 owner, dep_name = get_owner_name_from_github_url(url)
#                 if owner == 'buckaroo-pm' and dep_name not in self.buckaroo_parents:
#                     flag = False
#                     continue
#                 if dep_name in self.buckaroo_parents:
#                     owner, dep_name = self.buckaroo_parents[dep_name].split('/')[:2]
#                 continue
#             if line.startswith('version=') and flag:
#                 version = line.split('version=')[-1].strip('\"\'')
#                 if '=' in version:
#                     version = version.split('=')[-1]
#                 continue   
#             if line == '' and flag and dep_name:
#                 dep = Dependency(dep_name, version)
#                 dep.add_evidence(self.type, self.target, 'High')
#                 self.add_dependency(dep)
#                 flag = False
#                 continue
#         if flag and dep_name:
#             dep = Dependency(dep_name, version)
#             dep.add_evidence(self.type, self.target, 'High')
#             self.add_dependency(dep)
#             flag = False