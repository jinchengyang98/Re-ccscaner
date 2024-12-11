import re
import logging

import ccscanner.utils.cmakelists_parsing.parsing as cmp
from ccscanner.utils.utils import remove_lstrip, remove_rstrip

logging.basicConfig()
logger = logging.getLogger(__name__)

def cpm_func_analyzer(func_body):
## TODO: syntax https://github.com/cpm-cmake/CPM.cmake
    try:
        a = cmp.parse(func_body)
    except Exception as e:
        logger.error('cmake parsing error: '+self.target)
        logger.error(e)
    for i in a:
        if not isinstance(i, cmp._Command):
            continue
        if len(i.body) == 0:
            continue
        if len(i.body) == 1:
            dep_name, version = analyze_single_arg(i.body[0].contents)
        else:
            args = [arg.contents for arg in i.body]
            dep_name, version = analyzer_multi_arg(args)
        return dep_name, version
            

def analyze_single_arg(arg):
    arg = arg.strip('\'\"')
    shorthand = False
    if arg.startswith(('gh:', 'gl:')):
        shorthand = True
        arg = arg[3:]
    contents = re.split('@|#', arg)
    if len(contents) > 1:
        version = contents[1]
        if '@' in arg:
            for content in contents:
                if '@'+content in arg:
                    version = content
                    break
    else:
        version = None
    dep_name = contents[0].split('/')[-1]
    return dep_name, version


def analyzer_multi_arg(args):
    args_dict = {}
    for index in range(len(args))[::2]:
        if index + 1 == len(args):
            continue
        args_dict[args[index]] = args[index+1]
        
    dep_name = args_dict['name'] if 'name' in args_dict else None
    version = args_dict['version'] if 'version' in args_dict else None
    github_repo = args_dict['github_repository'] if 'github_repository' in args_dict else None
    git_tag = args_dict['git_tag'] if 'git_tag' in args_dict else None
    git_repo = args_dict['git_repository'] if 'git_repository' in args_dict else None

    if github_repo is not None:
        dep_name = github_repo.strip('\'\"').split('/')[-1]
    if github_repo is None and git_repo is not None:
        if '//github.com/' in git_repo:
            dep_name = remove_rstrip(git_repo, '.git').split('/')[-1]
    if version is None and git_tag is not None:
        version = git_tag
    return dep_name, version
