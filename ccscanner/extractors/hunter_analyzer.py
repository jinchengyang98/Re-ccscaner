import re
import logging

import ccscanner.utils.cmakelists_parsing.parsing as cmp
from ccscanner.utils.utils import remove_lstrip, remove_rstrip

logging.basicConfig()
logger = logging.getLogger(__name__)


def hunter_func_analyzer(func_body):
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
        dep_name = i.body[0].contents
        return dep_name
