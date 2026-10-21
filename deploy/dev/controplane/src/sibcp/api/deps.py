from collections.abc import AsyncIterator

from fastapi import Depends, HTTPException, Request, status 
from itsdangerous import BadSignature, SignatureExpired, URLSafeTimedSerializer 
from sqlalchemy.ext.asyncio import AsyncSession 

from ..settings import Settings

SESSION_COOKIE = "sibc_session"
SESSION_MAX_AGE = 12 * 3600 

def get_settings(request: Request) -> Settings:
    return request.app.state.settings 

async def get_session(request: Request) -> AsyncIterator[AsyncSession]:
    async with request.app.state.sessionmaker() as session:
        yield session
        
def serializer(settings: Settings) -> URLSafeTimedSerializer:
    return URLSafeTimedSerializer(settings.secret_key.get_secret_value(), salt="admin=session")

def current_admin(request: Request, settings: Settings = Depends(get_settings)) -> str:
    raw = request.cookies.get(SESSION_COOKIE)
    if not raw:
        raise HTTPException(status.HTTP_401_UNAUTHORIZED)
    try:
        return str(serializer(settings).loads(raw, max_age=SESSION_MAX_AGE))
    except (BadSignature, SignatureExpired):
        raise HTTPException(status.HTTP_401_UNAUTHORIZED) from None 
    