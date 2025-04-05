from core.dns_record import Record
from typing import Protocol

class DnsRegistry(Protocol):
    def register(self, record: Record): ...
    def remove(self, record: Record): ...
    def list(self) -> list[Record]: ...
