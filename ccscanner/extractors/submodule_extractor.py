import logging
import os
import re
from git import Repo, InvalidGitRepositoryError
import requests
import random
from ccscanner.extractors.extractor import Extractor
from ccscanner.utils.utils import read_js, save_js, read_lines, remove_rstrip
from ccscanner.extractors.dependency import Dependency

SUBMODS = 'ccscanner/data/submods.json'
logging.basicConfig()
logger = logging.getLogger(__name__)

KEYS = ['name', 'path', 'url', 'hexsha', 'branch_name', 'branch_path']
existing_submods = read_js(SUBMODS)

class SubmodExtractor(Extractor):
    def __init__(self, repo_path) -> None:
        super().__init__()
        self.type = 'gitsubmod'
        self.target = repo_path
        self.submods = []
    
    def run_extractor(self):
        self.submodule_extractor()

    def submodule_extractor(self):
        processed = 0
        try:
            submodules = Repo(self.target).submodules
        except InvalidGitRepositoryError:
            # target is not a valid git repo
            self.parse_submodule_file()
            processed = 1
        if processed == 0:
            for i in submodules:
                # skip if the submodule does not exist.
                # if not i.exists():
                #     continue
                r_item = {}
                for key in KEYS:
                    try:
                        r_item[key] = i.__getattr__(key)
                    except Exception as e:
                        continue
                # TODO add dependency
                self.submods.append(r_item)
        self.get_dep_name()

    def get_dep_name(self):
        for submod in self.submods:
            if len(submod) == 0:
                continue
            if 'url' in submod:
                dep_name = self.parse_url(submod['url'])
                if dep_name is None:
                    continue
            else:
                dep_name = submod['name'].split('.')[-1]
            if 'hexsha' in submod:
                version = submod['hexsha']
            else:
                version = None
            dep = Dependency(dep_name, version)
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)


    def parse_submodule_file(self):
        submodule_file = os.path.join(self.target, '.gitmodules')
        contents = read_lines(submodule_file)
        item = {}
        flag = 0
        for line in contents:
            if line.startswith('[submodule'):
                flag = 1
                if len(line) != 0:
                    self.submods.append(item)
                    item = {}
                item["name"] = re.search(r'\"(.*)\"', line).group(1)
            else:
                if flag == 0:
                    continue
                if '=' not in line:
                    continue
                key, value = line.split('=', 1)
                item[key.strip(' \t')] = value.strip(' \t')
        if len(item) > 0:
            self.submods.append(item)
                    

    def parse_url(self, url):
        if 'github.com' in url:
            url = remove_rstrip(url, '.git')
            owner, dep_name = url.split('github.com')[-1][1:].split('/')[:2]
            if owner+'@@'+dep_name in existing_submods:
                lang = existing_submods[owner+'@@'+dep_name]
                if lang not in ['C', 'C++']:
                    return None
                else:
                    return dep_name
            return dep_name
            # url = 'https://api.github.com/repos/{0}/{1}'.format(owner, dep_name)
            # try:
            #     info = requests.get(url.format(owner, dep_name), headers={
            #                 "Authorization": "token " + ACCESS_TOKENS[random.randint(0, len(ACCESS_TOKENS)-1)]}).json()
            #     if 'language' in info:
            #         existing_submods[owner+'@@'+dep_name] = info['language']
            #         save_js(existing_submods, SUBMODS)
            #         if info['language'] not in ['C', 'C++']:
            #             return None
            #         else:
            #             return dep_name
            # except Exception as e:
            #     print(e)
        else:
            return None