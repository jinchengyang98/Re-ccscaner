import os
import re
import json
import logging
from ccscanner.extractors.extractor import Extractor
from ccscanner.extractors.dependency import Dependency
from ccscanner.utils.utils import read_txt


# A re.VERBOSE regular expression to parse a PGP signed message in its parts.
# the re.VERBOSE flag allows for:
#  - whitespace is ignored except when in a character class or escaped
#  - anything after a '#' that is not escaped or in a character class is
#  ignored, allowing for comments
logging.basicConfig()
logger = logging.getLogger(__name__)

pgp_signed = re.compile(r"""
    # This capture group is optional because it will only be present in signed
    # cleartext messages
    (^-{5}BEGIN\ PGP\ SIGNED\ MESSAGE-{5}(?:\r?\n)
       (Hash:\ (?P<hashes>[A-Za-z0-9\-,]+)(?:\r?\n){2})?
       (?P<cleartext>(.*\r?\n)*(.*(?=\r?\n-{5})))(?:\r?\n)
    )?
    # Armor header line: capture the variable part of the magic text
    ^-{5}BEGIN\ PGP\ (?P<magic>[A-Z0-9 ,]+)-{5}(?:\r?\n)
    # Try to capture all the headers into one capture group.
    # If this doesn't match, m['headers'] will be None
    (?P<headers>(^.+:\ .+(?:\r?\n))+)?(?:\r?\n)?
    # capture all lines of the body, up to 76 characters long, including the
    # newline, and the pad character(s)
    (?P<body>([A-Za-z0-9+/]{1,76}={,2}(?:\r?\n))+)
    # capture the armored CRC24 value
    ^=(?P<crc>[A-Za-z0-9+/]{4})(?:\r?\n)
    # finally, capture the armor tail line, which must match the armor header
    # line
    ^-{5}END\ PGP\ (?P=magic)-{5}(?:\r?\n)?
     """, flags=re.MULTILINE | re.VERBOSE).search  # NOQA


class ControlExtractor(Extractor):
    """
    Parses Debian Control-File formats
    Exposes three attributes and one instance method:
    - self.raw_pkg_info   ==> Outputs a dictionary object with values after initial parse
    - self.clean_pkg_info ==> Outputs a dictionary object with only useful and clean values
    - self.pkg_names      ==> Outputs a list object with only the names of the packages in file
    - self.to_json_file() ==> Dumps dictionary outputs to a JSON file
    """

    def __init__(self, file):
        super().__init__()
        self.target = file
        self.file_text = self.__read_input(self.target)
        self.packages = self.file_text.strip('\n').split("\n\n")
        self.type = 'control'


    def run_extractor(self):
        pattern = r'\([^()]*\)'
        pattern2 = r'\<[^<>]*\>'
        operators = ['>=', '<=', '=', '<', '>']

        if len(self.packages[0]) > 0:
            self.raw_pkg_info = []
            for pkg in self.packages:
                try:
                    raw_info = self.__get_raw_info(pkg)
                    if raw_info:
                        self.raw_pkg_info.append(raw_info)
                except Exception as e:
                    continue
            if len(self.raw_pkg_info) == 0:
                return
            self.clean_pkg_info = [
                self.__get_clean_info(pkg) for pkg in self.raw_pkg_info
            ]
            self.pkg_names = [pkg["name"] for pkg in self.raw_pkg_info]
        else:
            return
            
        for pkg in self.raw_pkg_info:
            if 'build-depends' not in pkg['details']:
                continue
            for dep in pkg['details']['build-depends'].split(','):
                if '|' in dep:
                    dep_names = dep.split('|')
                else:
                    dep_names = [dep]
                for dep_item in dep_names:
                    # TODO add version extractor
                    dep_item = dep_item.strip()
                    dep_name = re.sub(pattern, '', dep_item)
                    dep_name = re.sub(pattern2, '', dep_name).strip()
                    version = dep_item.replace(dep_name, '').strip(' ()<>')
                    for operator in operators:
                        if operator in version:
                            version = version.split(operator)[-1].strip()
                            version = operator + version
                            break
                    dep = Dependency(dep_name, version)
                    dep.add_evidence(self.type, self.target, 'High')
                    self.add_dependency(dep)

    def to_json_file(
        self, outfile, names_only=False, raw=False
    ):
        """
        Dumps parsed data into JSON output file
        Attributes:
        - outfile= str, default: './datastore/dpkgs.json'
        - names_only= bool, default: False (if True supercedes other options, outputs list of names)
        - raw= bool, default: False (if True outputs raw parse)
        If both options are False, JSON will be based on clean package information
        """
        os.makedirs(os.path.dirname(outfile), exist_ok=True)
        try:
            if names_only:
                with open(outfile, "w") as f:
                    json.dump(self.pkg_names, f, indent=4)
            elif raw:
                with open(outfile, "w") as f:
                    json.dump(self.raw_pkg_info, f, indent=4)
            else:
                with open(outfile, "w") as f:
                    json.dump(self.clean_pkg_info, f, indent=4)
            logger.info("extracted information to JSON file")
        except:
            logger.exception("unable to write to file")

    # Private
    def __read_input(self, input_obj):
        """Ensures valid input type"""
        if type(input_obj) is not str:
            raise TypeError("input must be string or string path to file")
        elif os.path.exists(os.path.dirname(input_obj)):
            file_text = read_txt(input_obj)
            if self.__is_signed(file_text):
                file_text = self.__remove_signature(file_text).strip()
            return file_text
        else:
            return input_obj.strip()

    def __get_raw_info(self, text):
        """Parses a Debian control file and returns raw dictionary"""
        # Extract package keys and values
        split_regex = re.compile(r"^[A-Za-z-\d]+:\s", flags=re.MULTILINE)
        keys = [key[:-2].lower() for key in split_regex.findall(text)]
        values = [value.strip() for value in re.split(split_regex, text)[1:]]

        # Composing initial package info dict
        if len(values) > 0:
            pkg_name = values[0]
            pkg_type = 'source' if 'source' in keys else 'binary'
            pkg_details = dict(zip(keys[:], values[:]))
            pkg_dict = {"name": pkg_name, "type":pkg_type, "details": pkg_details}
            return pkg_dict
        else:
            # raise ValueError("file or text don't match Debian Control File schema")
            return None

            

    def __get_clean_info(self, raw_info):
        """Cleans up raw parsed package information and filters unneeded"""
        pkg_name = raw_info["name"]
        version = raw_info["details"].get("version")
        long_description = raw_info["details"].get("description")
        long_depends = raw_info["details"].get("depends")

        synopsis, description = self.__split_description(long_description)
        depends, alt_depends = self.__split_depends(long_depends)
        reverse_depends = self.__get_reverse_depends(pkg_name, self.raw_pkg_info)

        pkg_details = {
            "version": version,
            "synopsis": synopsis,
            "description": description,
            "depends": depends,
            "alt_depends": alt_depends,
            "reverse_depends": reverse_depends,
        }

        return {"name": pkg_name, "details": pkg_details}

    def __split_description(self, long_description):
        """Breaks down long descriptions into synopsis and description"""
        if long_description is not None:
            split_description = tuple(long_description.split("\n", maxsplit=1))
            synopsis = split_description[0]
            description = (
                re.sub(r"^\s", "", split_description[1], flags=re.MULTILINE)
                if 1 < len(split_description)
                else None
            )
        else:
            synopsis, description = None, None

        return (synopsis, description)

    def __split_depends(self, long_depends):
        """Breaks down dependencies text into two lists of dependencies and alternatives"""
        if long_depends is not None:
            depends_and_alt = long_depends.split(" | ")
            depends = depends_and_alt[0].split(", ")
            alt_depends = (
                depends_and_alt[1].split(", ") if 1 < len(depends_and_alt) else None
            )
        else:
            depends, alt_depends = None, None

        return (depends, alt_depends)

    def __get_reverse_depends(self, pkg_name, pkg_dict_list):
        """Gets the names of the packages that depend on the the specified one"""
        r_depends = []
        for pkg in pkg_dict_list:
            pkg_depends = pkg["details"].get("depends")
            if pkg_depends is not None:
                if pkg_name in pkg_depends:
                    r_depends.append(pkg["name"])

        return None if len(r_depends) == 0 else r_depends

    def __is_signed(self, text):
        """
        Return True if the text is likely PGP-signed.
        """
        if text and isinstance(text, str):
            text = text.strip()
            return text and (text.startswith('-----BEGIN PGP SIGNED MESSAGE-----')
                        and text.endswith('-----END PGP SIGNATURE-----'))
        return False


    def __remove_signature(self, text):
        """
        Return `text` stripped from a PGP signature if there is one. Return the
        `text` as-is otherwise.
        """
        if not self.__is_signed(text):
            return text

        signed = pgp_signed(text)
        if not signed:
            return text
        unsigned = signed.groupdict().get('cleartext')
        return unsigned



if __name__ == '__main__':
    print('ok')