import asyncio
import json 
import os
import tempfile 

import pytest

from sibcp.dataplane.base import PeerSpec 
from sibcp.dataplane.own_driver import OwnDataplaneDriver, UAPIError 

async def test_driver_against_fake_uapi_server():
    path = os.path.join(tempfile.mkdtemp(), "uapi.sock")
    peers = {}
    
    async def handle(reader, writer):
        req = json.loads(await reader.readline())
        if req["op"] == "set_peer":
            peers[req["peer"]["public_key"]] = {**req["peer"], "rx_bytes": 1, "tx_bytes": 2, "endpoint": "198.51.100.7:40000", "last_handshake_unix": 1700000000}
            resp = {"id": req["id"], "ok": True}
        elif req["op"] == "list_peers":
            resp = {"id": req["id"], "ok": True, "peers": list(peers.values())}
        else:
            resp = {"id": req["id"], "ok": False, "error": "unknown op"}
        writer.write(json.dumps(resp).encode() + b"\n")
        await writer.drain()
        writer.close()
        
    server = await asyncio.start_unix_server(handle, path)
    async with server:
        c = OwnDataplaneDriver(path)
        await c.set_peer(PeerSpec("k1", "10.77.0.2", 50))
        [st] = await c.list_peers()
        assert st.tunnel_ip == "10.77.0.2" and st.rx_bytes == 1 and st.last_handshake is not None 
        with pytest.raises(UAPIError):
            await c.remove_peer("k1")
            