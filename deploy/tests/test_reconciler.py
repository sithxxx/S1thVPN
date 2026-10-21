from controplane.src.sibcp.dataplane.base import PeerSpec 
from controplane.src.sibcp.dataplane.fake_driver import FakeDriver
from controplane.src.sibcp.api.reconciler import reconcile_once 

async def test_reconcile_add_update_remove():
    d = FakeDriver()
    await d.set_peer(PeerSpec("stale", "10.77.0.9", 50))
    await d.set_peer(PeerSpec("b", "10.77.0.3", 50))
    want = [PeerSpec("a", "10.77.0.2", 50), PeerSpec("b", "10.77.0.3", 25)]
    res = await reconcile_once(want, d)
    assert res.added == ["a"] and res.update == ["b"] and res.removed == ["stale"]
    assert (await reconcile_once(want, d)).added == []