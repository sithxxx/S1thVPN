from .base import PeerSpec, PeerStatus 

class FakeDriver:
    def __init__(self) -> None:
        self.peers: dict[str, PeerStatus] = {}
        
    async def ping(self) -> None:
        return None 
    
    async def set_peer(self, spec: PeerSpec) -> None:
        self.peers[spec.public_key] = PeerStatus(spec.public_key, spec.tunnel_ip, spec.rate_limit_mbps)
        
        async def remove_peer(self, public_key: str) -> None:
            self.peers.pop(public_key, None)
            
        async def list_peers(self) -> list[PeerStatus]:
            return list(self.peers.values())