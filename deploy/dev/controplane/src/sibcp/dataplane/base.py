from dataclasses import dataclass
from datetime import datetime
from typing import Protocol 

@dataclass(frozen=True)
class PeerSpec:
    public_key: str
    tunnel_ip: str 
    rate_limit_mbps: int 
    
@dataclass(frozen=True)
class PeerStatus:
    public_key: str 
    tunnel_ip: str 
    rate_limit_mbps: int 
    endpoint: str | None = None 
    rx_bytes: int = 0 
    tx_bytes: int = 0
    
class DataplaneDriver(Protocol):
    async def ping(self) -> None: ... 
    async def set_peer(self, spec: PeerSpec) -> None: ...
    async def remove_peer(self, public_key: str) -> None: ... 
    async def list_peers(self) -> list[PeerStatus]: ... 
    