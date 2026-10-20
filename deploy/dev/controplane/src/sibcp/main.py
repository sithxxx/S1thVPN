import asyncio 
import contextlib
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from fastapi import FastAPI
from .api import health, auth, invites, peers, users 
from .dataplane.base import DataplaneDriver, PeerSpec 
from .dataplane.own_driver import OwnDataplaneDriver
from .db import make_engine, make_sessionmaker
from .reconciler import Reconciler
from .services.devices import desired_peers
from .settings import Settings

def create_app(settings: Settings | None = None, driver: DataplaneDriver | None = None) -> FastAPI:
    settings = settings or Settings()
    
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        engine = make_engine(settings.database_url)
        maker = make_sessionmaker(engine)
        drv = driver or OwnDataplaneDriver(settings.uapi_socket)
        
        async def load() -> list[PeerSpec]:
            async with maker() as s:
                return await desired_peers(s)
            
        rec = Reconciler(load, drv, settings.reconcile_interval_s)
        app.state.settings, app.state.engine, app.state.sessionmaker = settings, engine, maker 
        app.state.driver, app.state.reconciler = drv, rec
        task = asyncio.create_task(rec.run())
        yield
        task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await task
        await engine.dispose()
    
    app = FastAPI(title="SibTunnel control plane", lifespan=lifespan)
    for r in (health.router, auth.roter, users.router, peers.router, invites.router):
        app.include_router(r)
    return app

def app() -> FastAPI:
    return create_app()