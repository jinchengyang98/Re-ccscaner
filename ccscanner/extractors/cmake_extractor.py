import re
import logging
import os

from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
import ccscanner.utils.cmakelists_parsing.parsing as cmp
from typing import NamedTuple
from ccscanner.extractors.utils import *
from ccscanner.utils.utils import read_txt, remove_lstrip, remove_rstrip
from ccscanner.extractors.conan_extractor import ConanExtractor
from ccscanner.extractors.cpm_analyzer import cpm_func_analyzer
from ccscanner.extractors.hunter_analyzer import hunter_func_analyzer

logging.basicConfig()
logger = logging.getLogger(__name__)
REGEX_OPTIONS = re.DOTALL | re.IGNORECASE | re.MULTILINE
PROJECT_VERSION = re.compile(
    "^\\s*set\\s*\\(\\s*VERSION\\s*\"([^\"]*)\"\\)", REGEX_OPTIONS)

SET_VAR_REGEX = re.compile(
    "^\\s*set\\s*\\(\\s*([a-zA-Z0-9_\\-]*)\\s+\"?([a-zA-Z0-9_\\-\\.\\$\\{\\}]*)\"?\\s*\\)", REGEX_OPTIONS)
INL_VAR_REGEX = re.compile("(\\$\\s*\\{([^\\}]*)\\s*\\})", REGEX_OPTIONS)
PROJECT = re.compile(
    "^ *project *\\([ \\n]*(\\w+)[ \\n]*.*?\\)", REGEX_OPTIONS)
SET_VERSION = re.compile(
    "^\\s*set\\s*\\(\\s*(\\w+)_version\\s+\"?([^\"\\)]*)\\s*\"?\\)", REGEX_OPTIONS)


# FIND_LIBRARY_SIGNATURE = ['names',
#   'hints', 'paths', 'path_suffixes', 'doc']
FIND_LIBRARY_OPTIONS = ['hints', 'paths', 'path_suffixes', 'doc',
                        'names_per_dir', 'no_cache', 'required',
                        'no_default_path', 'no_package_root_path', 'no_cmake_path', 'no_cmake_environment_path',
                        'no_system_environment_path', 'no_cmake_system_path', 'cmake_find_root_path_both', 'only_cmake_find_root_path', 'no_cmake_find_root_path']
PKG_CHECK_MODULES_OPTIONS = ['required', 'quiet', 'no_cmake_path',
                             'no_cmake_environment_path', 'imported_target', 'global']
OPERATORS = ['>=', '<=', '=', '<', '>']
CONAN_CMAKE_OPTIONS = ['generators', 'options', 'basic_setup', 'build', 'arch',
                       'conanfile', 'build_type',
                       'configuration_types', 'profile', 'profile_auto', 'conan_command', 'no_load', 'imports', 'install_folder', 'env', 'settings']


class Lib(NamedTuple):
    filenames: list
    version: str
    fromfile: str
    content: dict

# TODO: FetchContent
# https://cmake.org/cmake/help/latest/module/FetchContent.html
## TODO: ExternalProject
## https://cmake.org/cmake/help/latest/module/ExternalProject.html#module:ExternalProject, it is used by cpm.cmake

class CmakeExtractor(Extractor):
    def __init__(self, target) -> None:
        super().__init__()
        self.type = 'cmake'
        self.libs_found = []
        self.target = target


    def to_dict(self):
        return {'deps': self.deps, 'type': self.type, 'libs': self.libs_found}


    def run_extractor(self):
        if not self.target.endswith('CMakeLists.txt') and not self.target.endswith('.cmake'):
            logger.info("not a cmake file: " + self.target)
            return None
        else:
            logger.info("analyzing cmake file: " + self.target)
            self.cmake_analyzer()
    
    def cmake_analyzer(self):
        basename = os.path.basename(self.target).lower()
        if basename.startswith('find') and basename.endswith('.cmake'):
            dep_name = remove_lstrip(basename, 'find')
            dep_name = remove_rstrip(dep_name, '.cmake')
            dep = Dependency(dep_name, None)
            dep.add_evidence(self.type, self.target, 'High')
            self.add_dependency(dep)

        contents = read_txt(self.target)
        if contents is None:
            logger.error('reading errors: ' + self.target)
            return

        contents_replaced = CmakeExtractor.var_replace(contents).lower()
        self.find_library_analyzer(contents_replaced)
        self.find_package_analyzer(contents_replaced)
        self.get_deps_regex(contents_replaced)
        self.pkg_module_analyzer(contents_replaced)
        self.conan_cmake_analyzer(contents_replaced)
        self.check_library_exists_analyzer(contents_replaced)
        self.cpm_analyzer(contents_replaced)
        self.hunter_analyzer(contents_replaced)


    def get_deps_regex(self, contents):
        project_name = version = None

        projects = PROJECT.finditer(contents)
        count = 0
        p = next(projects, None)
        while(p):
            count += 1
            project_name = p.group(1)
            context = p.group(0).lower()
            version = None
            if 'version' in context:
                context = context.replace('\n', ' ')
                context_splited = context.split()
                if 'version' in context_splited:
                    index = context_splited.index('version')
                    version = context_splited[index+1].strip(')')
            if project_name and version:
                dep = Dependency(project_name, version)
                dep.add_evidence(self.type, context, 'High')
                self.add_dependency(dep)
                p = next(projects, None)
                return
            break

        versions = PROJECT_VERSION.finditer(contents)
        v = next(versions, None)
        while(v):
            group = v.group(1)
            version = parse_version(group, True)
            v = next(versions, None)
        if project_name:
            dep = Dependency(project_name, version)
            dep.add_evidence(self.type, context, '')
            self.add_dependency(dep)
        else:
            self.analyze_version_command(contents)

    def get_func_body(self, pattern, contents):
        index_iter = re.finditer(pattern, contents)
        funcs = []
        pattern = r'\#.*\n'
        for index in index_iter:
            cursor = index.start()
            if not self.check_comment(cursor, contents):
                continue
            func_body = ''
            left_count = 0
            flag = True
            cursor_over = 0
            while(flag):
                func_body += contents[cursor]
                if contents[cursor] == '(':
                    left_count += 1
                if contents[cursor] == ')':
                    left_count -= 1
                    if left_count == 0:
                        flag = False
                cursor += 1
                if cursor > len(contents):
                    cursor_over = 1
                    break
            if cursor_over == 0:
                func_body = re.sub(pattern, '\n', func_body)
                funcs.append(func_body)
        return funcs

    def check_comment(self, cursor, contents):
        while(cursor):
            if contents[cursor] == '#':
                return False
            elif contents[cursor] == '\n':
                return True
            else:
                cursor -= 1

    def find_library_analyzer(self, contents):
        # pattern = '(find_library\s*\([^)]*\))'
        pattern1 = 'find_library\s*\('
        pattern2 = 'find_program\s*\('
        funcs1 = self.get_func_body(pattern1, contents)
        funcs2 = self.get_func_body(pattern2, contents)
        funcs = funcs1 + funcs2
        for func_body in funcs:
            try:
                a = cmp.parse(func_body)
            except Exception as e:
                logger.error('cmake parsing error: '+self.target)
                logger.error(e)
                continue
            for i in a:
                if not isinstance(i, cmp._Command):
                    continue
                if i.name.lower() == 'find_library' or i.name.lower() == 'find_program':
                    if len(i.body) < 2:
                        continue
                    if i.body[1].contents.lower() != 'names' and i.body[1].contents.lower() not in FIND_LIBRARY_OPTIONS:
                        lib = Lib([i.body[1].contents], '',
                                  self.target, func_body)
                        self.libs_found.append(lib._asdict())
                    if i.body[1].contents.lower() == 'names':
                        names = []
                        for arg in i.body[2:]:
                            if arg.contents.lower() in FIND_LIBRARY_OPTIONS:
                                break
                            names.append(arg.contents)
                        lib = Lib(names, '', self.target, func_body)
                        self.libs_found.append(lib._asdict())

    def find_package_analyzer(self, contents):
        pattern = 'find_package\s*\('
        funcs = self.get_func_body(pattern, contents)

        for func_body in funcs:
            try:
                a = cmp.parse(func_body)
            except Exception as e:
                logger.error('cmake parsing error: '+self.target)
                logger.error(e)
                continue
            for i in a:
                if not isinstance(i, cmp._Command):
                    continue
                if i.name.lower() == 'find_package':
                    if len(i.body) == 0:
                        continue
                    dep_name = i.body[0].contents
                    if len(i.body) == 1:
                        version = None
                    else:
                        version = parse_version(i.body[1].contents, True)
                    dep = Dependency(dep_name, version)
                    dep.add_evidence(self.type, func_body, 'High')
                    self.add_dependency(dep)


    def check_library_exists_analyzer(self, contents):
        pattern = 'check_library_exists\s*\('
        funcs = self.get_func_body(pattern, contents)

        for func_body in funcs:
            try:
                a = cmp.parse(func_body)
            except Exception as e:
                logger.error('cmake parsing error: '+self.target)
                logger.error(e)
                continue
            for i in a:
                if not isinstance(i, cmp._Command):
                    continue
                if i.name.lower() == 'check_library_exists':
                    dep_name = i.body[0].contents
                    dep = Dependency(dep_name, None)
                    dep.add_evidence(self.type, func_body, 'High')
                    self.add_dependency(dep)
                    


    @staticmethod
    def parse_pkg_version(name):
        version = opperator_op = None
        for operator in OPERATORS:
            if operator in name:
                name, version = name.split(operator)[:2]
                opperator_op = operator
                break
        if version is not None:
            return name, version, opperator_op
        if '-' in name:
            version = parse_version(name.split('-')[-1], True)
            if version is not None:
                name = '-'.join(name.split('-')[:-1])
                opperator_op = '='
        return name, version, opperator_op

    def pkg_module_analyzer(self, contents):
        pattern1 = 'pkg_check_modules\s*\('
        pattern2 = 'pkg_search_module\s*\('
        funcs1 = self.get_func_body(pattern1, contents)
        funcs2 = self.get_func_body(pattern2, contents)
        funcs = funcs1 + funcs2
        for func_body in funcs:
            try:
                a = cmp.parse(func_body)
            except Exception as e:
                logger.error('cmake parsing error: '+self.target)
                logger.error(e)
                continue
            for i in a:
                if not isinstance(i, cmp._Command):
                    continue
                if i.name.lower() == 'pkg_check_modules' or i.name.lower() == 'pkg_search_module':
                    if len(i.body) == 0:
                        continue
                    for arg in i.body[1:]:
                        name = arg.contents
                        if name in PKG_CHECK_MODULES_OPTIONS:
                            continue
                        name, version, opperator_op = CmakeExtractor.parse_pkg_version(name)
                        dep = Dependency(name, version, opperator_op)
                        dep.add_evidence(self.type+'::pkg', func_body, 'High')
                        self.add_dependency(dep)

    def conan_cmake_analyzer(self, contents):
        pattern1 = 'conan_cmake_run\s*\('
        pattern2 = 'conan_cmake_configure\s*\('
        funcs1 = self.get_func_body(pattern1, contents)
        funcs2 = self.get_func_body(pattern2, contents)
        funcs = funcs1 + funcs2
        for func_body in funcs:
            try:
                a = cmp.parse(func_body)
            except Exception as e:
                logger.error('cmake parsing error: '+self.target)
                logger.error(e)
                continue
            for i in a:
                if not isinstance(i, cmp._Command):
                    continue
                if i.name.lower() == 'conan_cmake_run' or i.name.lower() == 'conan_cmake_configure':
                    if len(i.body) == 0:
                        continue
                    key_flag = 0
                    for arg in i.body:
                        if arg.contents.lower() == 'requires' or arg.contents.lower() == 'build_requires':
                            key_flag = 1
                            continue
                        if key_flag == 1:
                            if '/' in arg.contents and arg.contents.lower() not in CONAN_CMAKE_OPTIONS:
                                name, version = ConanExtractor.parse_conan_package(
                                    arg.contents)
                                dep = Dependency(name, version, '=')
                                dep.add_evidence(self.type+"::conan", func_body, 'High')
                                self.add_dependency(dep)
                            if arg.contents.lower() in CONAN_CMAKE_OPTIONS:
                                key_flag = 0

    # collect all variables in cmake files and delete assignment loop.
    def cpm_analyzer(self, contents):
        pattern1 = 'cpmaddpackage\s*\('
        pattern2 = 'cpmfindpackage\s*\('
        funcs1 = self.get_func_body(pattern1, contents)
        funcs2 = self.get_func_body(pattern2, contents)
        funcs = funcs1 + funcs2
        for func_body in funcs:
            dep_name, version = cpm_func_analyzer(func_body)
            if dep_name is None:
                continue
            dep = Dependency(dep_name, version)
            dep.add_evidence(self.type+'::cpm', func_body, 'High')
            self.add_dependency(dep)


    def hunter_analyzer(self, contents):
        pattern = 'hunter_add_package\s*\('
        funcs = self.get_func_body(pattern, contents)
        for func_body in funcs:
            dep_name  = hunter_func_analyzer(func_body)
            if dep_name is None:
                continue
            dep = Dependency(dep_name, None)
            dep.add_evidence(self.type+'::hunter', func_body, 'High')
            self.add_dependency(dep)



    @staticmethod
    def collect_var(contents):
        vars_name2value = {}
        vars = SET_VAR_REGEX.finditer(contents)
        for var in vars:
            vars_name2value[var.group(1)] = var.group(2)

        # delete var loop, A = ${B} and B= ${A}. code will not stop in loop like this.
        to_delete_keys = []
        for var in vars_name2value:
            value = INL_VAR_REGEX.search(vars_name2value[var])
            if value:
                value = value.group(2)
            else:
                continue
            if value in vars_name2value:
                value2 = INL_VAR_REGEX.search(vars_name2value[value])
                if value2:
                    value2 = value2.group(2)
                else:
                    continue
            else:
                continue
            if var == value2:
                to_delete_keys.append(var)
                to_delete_keys.append(value)
        for key in set(to_delete_keys):
            del vars_name2value[key]

        return vars_name2value

    # TODO: pkg_search_module, pkg_check_module, conan_cmake_run
    # replace variables with their values in cmake files

    @staticmethod
    def var_replace(contents):
        vars = CmakeExtractor.collect_var(contents)
        inl_vars = INL_VAR_REGEX.finditer(contents)
        contents_replacer = contents
        r = next(inl_vars, None)
        while(r):
            least_one = False
            if r.group(2) in vars:
                if r.group(2) not in vars[r.group(2)]:
                    contents_replacer = contents_replacer.replace(
                        r.group(1), vars[r.group(2)])
                    inl_vars = INL_VAR_REGEX.finditer(contents_replacer)
                    least_one = True
            r = next(inl_vars, None)
            while(r):
                if r.group(2) in vars:
                    if r.group(2) not in vars[r.group(2)]:
                        contents_replacer = contents_replacer.replace(
                            r.group(1), vars[r.group(2)])
                        inl_vars = INL_VAR_REGEX.finditer(contents_replacer)
                        least_one = True
                r = next(inl_vars, None)
            if not least_one:
                break
            inl_vars = INL_VAR_REGEX.finditer(contents_replacer)
            r = next(inl_vars, None)
        contents_replaced = contents_replacer
        return contents_replaced

    def analyze_version_command(self, contents):
        vers = SET_VERSION.finditer(contents)

        count = 0
        product = version = None
        v = next(vers, None)
        while(v):
            count += 1
            product = v.group(1)
            version = v.group(2)
            if product.startswith('_'):
                product = product[1:]
            if product.lower().endswith('lib'):
                product = "lib" + product.lower()[0:-3]
            version = parse_version(version, True)
            dep = Dependency(product, version)
            dep.add_evidence(self.type, v.group(0), 'Low')
            self.add_dependency(dep)
            v = next(vers, None)