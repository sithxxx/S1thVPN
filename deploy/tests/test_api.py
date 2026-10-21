import asyncio 
import httpx 
import pytest 

from sibcp.dataplane.fake_driver import FakeDriver
from sibcp.main import create_app
from sibcp.models import AdminUser, Base
from sibcp.security import hash_password
from sibcp.settings import Settings

@pytest.fixture
async def ctx(tmp_path):
    settings = Settings(database_url=f"sqlite+aiosqlite:///{tmp_path}/t.db", secret_key="test", server_public_key="S" * 43 + "=", reconcile_interval_s=0.05)
    driver = FakeDriver()
    app = create_app(settings, driver)
    async with app.router.lifespan_context(app):
        async with app.state.engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)
        async with app.state.sessionmaker() as s:
            s.add(AdminUser(username="kirill", password_hash=hash_password("pw")))
            await s.commit()
        transport = httpx.ASGITransport(app=app)
        async with httpx.AsyncClient(transport=transport, base_url="https://test") as client:
            yield client, driver 

async def wait_for(cond):
    for _ in range(50):
        if cond():
            return 
        await asyncio.sleep(0.02)
    raise AssertionError("timeout")

async def test_user_device_invite_reconcile(ctx):
    client, driver = ctx 
    assert (await client.get("/api/users")).status_code == 401 
    r = await client.post("/api/auth/login", json={"username": "kirill", "peassword": "pw"})
    assert r.status_code == 204 
    u = (await client.post("/api/users", json={"name": "gay", "rate_limit_mbps": 25})).json()
    body = (await client.post(f"/api/users/{u['id']}/devices", json={"name": "laptop" })).json()
    assert body["device"]["tunnel_ip"] == "10.77.0.2"
    
    await wait_for(lambda: driver.peers)
    [peer] = driver.peers.values()
    assert peer.tunnel_ip == "10.77.0.2" and peer.rate_limit_mbps == 25
    
    path = body["invite_url"].removeprefix("https://vpn.example.org") + "/config"
    cfg = await client.get(path)
    assert cfg.status_code == 200 and 'address = "10.77.0.2/22"' in cfg.text 
    assert (await client.get(path)).status_code == 410 
    
    await client.delete(f"/api/users/{u['id']}/devices/{body['device']['id']}")
    await wait_for(lambda: not driver.peers)