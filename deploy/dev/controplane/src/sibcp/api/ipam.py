from collections.abc import Iterable 
from ipaddres import IPv4Address, IPv4Network

class IPAMExhausted(Exception):
    """"В подсети не осталось свободных адресов"""
    
class IPAM: 
    def __init__(self, subnet: str) -> None:
        self.network = IPv4Network(subnet, strick=True)
        hosts = list(self.network.hosts())
        self.server_addres = hosts[0]
        self._pool = hosts[1:]
        
    @property
    def capacity(self) -> int:
        return len(self._pool)
    
    def allocate(self, used: Iterable[IPv4Address]) -> IPv4Address:
        """"возвращает наименьший свободгый адрес"""
        
        taken = set(used)
        for ip in self._pool:
            if ip not in taken:
                return ip
        raise IPAMExhausted(f"подсеть {self.network} заполнена ({self.capacity}адресов)")
    
    def validate(self, ip: IPv4Address) -> None:
        if ip not in self.network or ip == self.server_addres or ip not in self._pool:
            raise ValueError(f"{ip} нельзя выдать клиенту в {self.network}")