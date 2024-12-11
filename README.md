# Towards Understanding Third-party Library Dependency in C/C++ Ecosystem

## Dataset
All extracted dependencies and ground truth are available at: https://figshare.com/s/9e2fd7a1389af8266bfe

Detection targets can be downloaded from official GitHub repositories or official websites.
##  CCScanner
CCScanner is a dependency scanner for C/C++ software. More details can be seen in our paper, [Towards Understanding Third-party Library Dependency in C/C++ Ecosystem](https://arxiv.org/abs/2209.02575).

You can try to use it by running "scanner.py" or install the pip package "ccscanner".

**Please note that the code of Centris module is not contained in this repo.**

Centris is available at: https://github.com/WOOSEUNGHOON/Centris-public.

results.json contains the extracted dependencies from "tests/test_data".

### Install
CCScanner is written using Python3.
Install dependencies.
```·
pip install json5 bs4 GitPython lxml requests
```
Or
```·
pip install -r requirements.txt
```

### Usage
run command:
```
cd ccscanner
python ccscanner/scanner.py -d $directory_to_scan -t $results_json_file
```
All results will be saved to a json files. The default path to save results is ```./results.json```.

```Deps``` field in results is all extracted dependencies.

### Pip package
We have released a pip package. You can try to use it.

```
pip install ccscanner
$ ccscanner_print -d ${directory to scan} -t ${file to save}
```

### Test
```tests``` directory contains some configuration files in real-world applications to test CCScanner.

CCScanner will be continuously maintained in future. It is welcomed if you have any issues or want to join us to improve it.

