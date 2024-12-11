import logging
logging.basicConfig()
logger = logging.getLogger(__name__)

class Extractor(object):
    def __init__(self) -> None:
        super().__init__()
        self.deps = []
        self.type = ''

    def add_dependency(self, dep):
        self.deps.append(dep.to_dict())
    
    def get_deps(self):
        return self.deps

    def run_extractor(self):
        logging.info("start running extractor...")

    def to_dict(self):
        return {'deps': self.deps, 'type': self.type}