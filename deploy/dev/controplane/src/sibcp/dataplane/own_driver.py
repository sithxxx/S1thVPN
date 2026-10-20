import asyncio 
import itertools 
import json 
from datetime import UTC, datetime 
from typing import Any 
from .base import PeerSpec, PeerStatus 

class UAPIError(Exception):
    def __init__(self, code: str) -> None:
        super().__init__(code)
        self.code = code 
        
class OwnDataplaneDriver:
    """Драйвер нашего Go-датаплейна: UAPI (JSON-строки) через unix-сокет"""
    
    def __init__(self, socket_path: str, timeout: float = 0.5) -> None:
        self.socket_path = socket_path 
        self.timeout = timeout 
        self._ids = itertools.count(1)
    
    async def _request(self, op: str, **params: Any) -> dict[str, Any]:
        req_id = next(self._ids)
        reader, writer = await asyncio.wait_for(
            asyncio.open_unix_connection(self.socket_path), self.timeout 
        )
        try:
            writer.write(json.dumps({"id": req_id, "op": op, **params}).encode() + b"\n")
            await writer.drain()
            line = await asyncio.wait_for(reader.readline(), self.timeout)
        finally: 
            writer.close()
            await writer.wait_closed()
        if not line:
            raise UAPIError("connection_closed")
        resp: dict[str, Any] = json.loads(line)
        if resp.get("id") != req_id:
            raise UAPIError("id_mismatch")
        if not resp.get("ok"):
            raise UAPIError(str(resp.get("error", "unknown")))
        return resp 
    async def ping(self) -> None:
        await self._request("ping")
        
    async def set_peer(self, spec: PeerSpec) -> None:
        await self._request("set_peer", peer={
            "public_key": spec.public_key, 
            "tunnel_ip": spec.tunnel_ip, 
            "rate_limit_mbps": spec.rate_limit_mbps,
        })
    
    async def remove_peer(self, public_key: str) -> None:
        await self._request("remove_peer", public_key=public_key)
        
    async def list_peers(self) -> list[PeerStatus]:
        resp = await self._request("list_peers")
        result = []
        for p in resp.get("peers", []):
            ts = p.get("last_handshake_unix") or 0
            result.append(PeerStatus(
                public_key=p["public_key"],
                tunnel_ip=p["tunnel_ip"],
                rate_limit_mbps=p["rate_limit_mbps"],
                endpoint=p.get("endpoint") or None,
                last_handshake=datetime.fromtimestamp(ts, UTC) if ts else None,
                rx_bytes=p.get("rx_bytes", 0),
                tx_bytes=p.get("tx_bytes", 0),
            ))
        return result