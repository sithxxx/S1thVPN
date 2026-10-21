from fastapi import APIRouter, Depends, HTTPException, Response, status 
from pydantic import BaseModel 
from sqlalchemy import select 
from sqlalchemy.ext.asyncio import AsyncSession 

from ..models import AdminUser 
from ..security import verify_password 
from ..settings import Settings 
from .deps import SESSION_COOKIE, SESSION_MAX_AGE, get_session, get_settings, serializer 

router = APIRouter(prefix="/api/auth", tags=["auth"])

class LoginIn(BaseModel):
    username: str
    password: str
    
@router.post("/login", status_code=204)
async def login(body: LoginIn, response: Response, 
                session: AsyncSession = Depends(get_session), 
                settings: Settings = Depends(get_settings)) -> None:
    admin = await session.scalar(select(AdminUser).where(AdminUser.username == body.username))
    if admin is None or not verify_password(admin.password_hash, body.password):
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "неверный логин или пароль")
    response.set_cookie(SESSION_COOKIE, serializer(settings).dumps(admin.username), 
                        max_age=SESSION_MAX_AGE, httponly=True, secure=settings.cookie_secure, samesite="strict")
    
@router.post("/logout", status_code=204)
async def logout(response: Response) -> None:
    response.delete_cookie(SESSION_COOKIE)