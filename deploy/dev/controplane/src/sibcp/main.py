from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from fastapi import FastAPI
from .api import health 
from .db import make_engine, make_sessionmaker
from .settings import Settings

def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or Settings()
    
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        engine = make_engine(settings.database_url)
        app.state.settings, app.state.engine = settings, engine
        app.state.sessionmaker = make_sessionmaker(engine)
        yield
        await engine.dispose()
        
    app = FastAPI(title="SibTunnel control plane", lifespan=lifespan)
    app.include_router(health.router)
    return app

def app() -> FastAPI:
    return create_app()