import json
import json5
import re
import csv
import sys
import logging
import os
from bs4 import BeautifulSoup

logging.basicConfig()
logger = logging.getLogger(__name__)


def save_js(content, path):
    with open(path, 'w') as save_f:
        json.dump(content, save_f)

def add_line(line, path):
    with open(path, 'a') as save_f:
        save_f.write(line)


def read_js(path):
    try:
        with open(path, 'r', errors='ignore') as read_f:
            content = json.load(read_f)
    except:
        try:
            with open(path, 'r', encoding='latin-1') as read_f:
                content = json.load(read_f)
        except:
            return None
    return content

def read_json5(path):
    with open(path) as data_file:    
        content = json5.load(data_file)
    return content

    # try:
    #     with open(path, 'r') as read_f:
    #         content = json.load(read_f)
    #     return content
    # except:
    #     logger.error("read error: " + path)
    #     return None
    

def read_xml(path):
    try:
        with open(path, 'r', errors='ignore') as f:
            data = f.read()
    except:
        try:
            with open(path, 'r', encoding='latin-1') as f:
                data = f.read()
        except:
            return None

    xml_data = BeautifulSoup(data, "xml")
    return xml_data


def read_txt(txt):
    try:
        with open(txt, 'r', errors='ignore') as load_f:
            content = load_f.read()
    except:
        try:
            with open(txt, 'r', encoding='latin-1') as load_f:
                content = load_f.read()
        except:
            return None
    return content


def read_csv(csv_file):
    csv.field_size_limit(sys.maxsize)
    csvfile = open(csv_file)
    csv_content = csv.reader(csvfile, delimiter=',', skipinitialspace=True)
    return csv_content


def read_lines(text):
    try:
        with open(text, 'r', errors='ignore') as load_f:
            lines = [line.rstrip() for line in load_f.readlines()]
    except:
        try:
            with open(text, 'r', encoding='latin-1') as load_f:
                lines = [line.rstrip() for line in load_f.readlines()]
        except:
            return None
    return lines


def get_lib_names():
    current_path = os.getcwd()
    names_filepath = os.path.join(current_path, '../../data/cc_libs/all_libs.txt')
    names = read_lines(names_filepath)
    return names


def remove_lstrip(s, substring):
    if not s.startswith(substring):
        return s
    else:
        return s[len(substring):]


def remove_rstrip(s, substring):
    if not s.endswith(substring):
        return s
    else:
        return s[:len(substring)*-1]


def count_empty_dir(root, print_emtpy_dir = False):
    dirs = os.listdir(root)
    count = 0
    for dir in dirs:
        if len(os.listdir(os.path.join(root, dir))) == 0:
            if print_emtpy_dir:
                print(dir)
            count += 1
    return count
    

def get_func_body(pattern, contents):
        index_iter = re.finditer(pattern, contents)
        funcs = []
        for index in index_iter:
            cursor = index.start()
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
                funcs.append(func_body)
        return funcs

def get_unified_name(name):
    VERSION_SUFFIX_PATTERN = '[._-]?\d+(\.\d+){1,6}([._-]?(snapshot|release|final|alpha|beta|rc$|[a-zA-Z]{1,3}[_-]?\d{1,8}))?$'
    if ' ' in name:
        name = name.split(' ')[0]
    if any(i in name for i in [':', '"', '[', ']', '$', '(', ')', '{', '}']):
        return None
    if '@@' in name:
        name = name.split('@@')[-1]
    name = remove_lstrip(name.lower(), 'lib')
    while(name.endswith(('_find', '_major', '_minor', '_min', '_patchlevel', '_patch'))):
        name = '_'.join(name.split('_')[:-1])
    for suffix in ['-dev', '_dev', '-src', '_src']:
        name = remove_rstrip(name, suffix)
    version = re.search(VERSION_SUFFIX_PATTERN, name)
    if version is not None:
        name = name.replace(version.group(0), '').strip('-_.')
    return name

if __name__ == '__main__':
    read_js('test/test_data/error.json')