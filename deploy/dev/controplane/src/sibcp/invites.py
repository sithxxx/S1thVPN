from fastapi import APIRouter, Depends, HTTPException 
from fastapi.responses import PlainTextResponse 
from sqlalchemy.ext.asyncio import AsyncSession 

from ..services.devices import redeem_invite 
from .deps import get_session 

router = APIRouter(tags=["invites"]) 

@router.get("/i/{token}/config", response_class=PlainTextResponse)
async def download_config(token: str, session: AsyncSession = Depends(get_session)) -> PlainTextResponse:
    config =await redeem_invite(session, token)
    if config is None:
        raise HTTPException(410, "ссылка недействительна или уже использована")
    return PlainTextResponse(config, headers={
        "Content-Disposition": 'attachment; filename="sibtunnel.toml"',
        "Cache-Control": "no-store", 
    })
    