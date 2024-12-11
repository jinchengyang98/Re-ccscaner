OPERATORS = ['>=', '<=', '=', '<', '>']


def parse_version_str(version_str):
    ## version_str: "xxx>=1.1" or ">=1.1"
    version_str = version_str.replace(' ', '')
    name = version = None
    for operator in OPERATORS:
        if operator in version_str:
            name, version = version_str.split(operator)[:2]
            opperator_op = operator
            break
    return name, version, opperator_op