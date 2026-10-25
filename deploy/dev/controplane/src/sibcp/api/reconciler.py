import asyncio 
import logging 
from collections.abc import Awaitable, Callable, Iterable
from dataclasses import dataclass, field 

from .dataplane.base import DataplaneDriver, PeerSpec 

log = logging.getLogger(__name__)

@dataclass 
class ReconcileResult:
    added: list[str] = field(default_factory=list)
    updated: list[str] = field(default_factory=list)
    removed: list[str] = field(default_factory=list)
    
async def reconcile_once(desired: Iterable[PeerSpec], driver: DataplaneDriver) -> ReconcileResult:
    actual = {p.public_key: p for p in await driver.list_peers()}
    want = {p.public_key: p for p in desired}
    res = ReconcileResult()
    for key, spec in want.items():
        cur = actual.get(key)
        if cur is None:
            await driver.set_peer(spec)
            res.added.append(key)
        elif (cur.tunnel_ip, cur.rate_limit_mbps) != (spec.tunnel_ip, spec.rate_limit_mbps):
            await driver.set_peer(spec)
            res.updated.append(key)
    for key in actual.keys() - want.keys():
        await driver.remove_peer(key)
        res.removed.append(key)
    return res 

class Reconciler:
    def __init__(
        self,
        load_desired: Callable[[], Awaitable[list[PeerSpec]]],
        driver: DataplaneDriver,
        interval: float = 10.0,
    ) -> None:
        self._load = load_desired 
        self._driver = driver
        self._interval = interval 
        self._wake = asyncio.Event()
        
    def trigger(self) -> None:
        self._wake.set()
    async def run(self) -> None:
        while True:
            try: 
                res = await reconcile_once(await self._load(), self._driver)
                if res.added or res.updated or res.removed:
                    log.info("reconciled: +%d ~%d -%d", len(res.added), len(res.updated), len(res.removed))
            except Exception: 
                log.exception("reconcile failed")
            try:
                await asyncio.wait_for(self._wake.wait(), self._interval)
            except TimeoutError:
                pass
            self._wake.clear()
            try:
                desired = await self._load()
                metrics.PEERS_DESIRED.set(len(desired))
                res = await reconcile_once(desired, self._driver)
                metrics.RECONCILE_RUNS.inc()
                for kind in ("added", "updated", "removed"):
                    
metrics.RECONCILE_CHANGES.labels(kind).inc(len(getattr(res, kind)))
                ...
            except Exception:
                metrics.RECONCILE_ERRORS.inc()
                log.exception("reconcile failed")
