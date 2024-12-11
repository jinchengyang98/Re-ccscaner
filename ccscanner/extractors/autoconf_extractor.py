import re
import os
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_txt, get_func_body

PACKAGE_VAR = re.compile("PACKAGE_(.+?)='(.*?)'", re.DOTALL | re.IGNORECASE)
param = "\\s*\\[{0,2}(.+?)\\]{0,2}"
sepParam = "\\s*," + param
AC_INIT_PATTERN = re.compile("AC_INIT\\(%s%s(%s)?(%s)?(%s)?\\s*\\)" % (param, sepParam, sepParam, sepParam, sepParam), re.DOTALL| re.IGNORECASE)

KEY_FILES = ['configure', 'configure.in', 'configure.ac']


class AutoconfExtractor(Extractor):
    def __init__(self, file_path) -> None:
        super().__init__()
        self.type = 'autoconf'
        self.target = file_path

    def run_extractor(self):
        self.autoconf_extractor(self.target)


    def autoconf_extractor(self, file_path):
        file_name = os.path.basename(file_path)
        product = version = vendor = None
        # if not file_name.lower().startswith('configure'):
        #     return None, None
        if file_name.lower() in KEY_FILES:
            contents = read_txt(file_path)
            if file_name.lower() == 'configure':
                self.extract_from_configure(contents)
            else:
                self.extract_from_confin_confac(contents)
            self.parse_funcs(contents)


    def extract_from_configure(self, contents):
        package_vars = PACKAGE_VAR.finditer(contents)
        package_var = next(package_vars, None)
        product = version = vendor = None
        while(package_var):
            var = package_var.group(1)
            value = package_var.group(2)
            if value:
                if var.endswith("NAME"):
                    product = value
                elif var == 'VERSION':
                    version = value
                elif var == "BUGREPORT":
                    vendor = value
                elif var == "URL":
                    vendor = value
                #TODO: add vendor
            package_var = next(package_vars, None)
        if product is not None:
            dep = Dependency(product, version)
            dep.add_evidence(self.type, self.target, '')
            self.add_dependency(dep)
        

    def extract_from_confin_confac(self, contents):
        iters = AC_INIT_PATTERN.finditer(contents)
        iter = next(iters, None)
        product = version = vendor = None
        while(iter):
            # TODO: iter.group(5) and iter.group(7) are ignored in current version.
            product = iter.group(1)
            if ")" not in product:
                version = iter.group(2)
                # TODO 'AC_INIT(\n [libewf],\n [20220130],\n [joachim.metz@gmail.com])\n'
                # extracted vendor is ',\n [joachim.metz@gmail.com]'
                # todo: fix this issue
                vendor = iter.group(3)
                dep = Dependency(product, version)
                dep.add_evidence(self.type, iter.group(0), '')
                self.add_dependency(dep)
            iter = next(iters, None)
    
    def parse_funcs(self, contents):
        pattern = 'AC_CHECK_LIB\('
        funcs = get_func_body(pattern, contents)
        for func in funcs:
            args = func.replace('AC_CHECK_LIB', '').strip('()')
            dep_name = args.split(',')[0].strip().strip('[]')
            if dep_name is not None:
                dep = Dependency(dep_name, None)
                dep.add_evidence(self.type, self.target, 'High')
                self.add_dependency(dep)
