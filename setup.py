import setuptools

with open("README_pip.md", "r", encoding="utf-8") as read_f:
    long_description = read_f.read()
    
with open('ccscanner/requirements.txt', 'r') as read_f:
    requires_list = read_f.readlines()
requires_list = [i.strip() for i in requires_list]

setuptools.setup(
    name="ccscanner",
    version="0.1.10",
    author="lkp",
    author_email="",
    url="https://github.com/lkpsg/ccscanner",
    description="A SBOM scanner for C/C++",
    long_description=long_description,
    long_description_content_type="text/markdown",
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    packages = setuptools.find_packages(),
    python_requires=">=3.6",
    entry_points = {
        'console_scripts': [
            'ccscanner_print = ccscanner.scanner:main',
        ]
    },
    install_requires=requires_list
)