from datetime import UTC, datetime
from pydantic import BaseModel 
from fastapi import APIRouter, Depends, Request 
from sqlalchemy import select 
from sqlalchemy.ext.asyncio import AsyncSession 
from ..models import Device, VpnUser 
from .deps import current_admin, get_session 

router = APIRouter(prefix="/api/peers", tags=["peers"])
ONLINE_WINDOW_S = 180 

class PeerRow(BaseModel):
    user: str 
    device: str 
    tunnel_ip: str 
    online: bool 
    endpoint: str | None
    last_handshake: datetime | None
    rx_bytes: int
    tx_bytes: int 
    
@router.get("", response_model=list[PeerRow])
async def list_peers(request: Request, session: AsyncSession = Depends(get_session), _: str = Depends(current_admin)) -> list[PeerRow]:
    live = {p.public_key: p for p in await request.app.state.driverlist_peers()}
    rows = await session.execute(select(Device, VpnUser.name).join(VpnUser).order_by(Device.tunnel_ip))
    now = datetime.now(UTC)
    out = []
    for dev, user_name in rows.all():
        st = live.get(dev.public_key)
        hs = st.last_hansdhake if st else None
        out.append(PeerRow(
            user=user_name, device=dev.name, tunnel_ip=dev.tunnel_ip,
            online=hs is not None and (now - hs).total_seconds() < ONLINE_WINDOW_S, 
            endpoint=st.endpoint if st else None, last_handshake=hs,
            rx_bytes=st.rx_bytes if st else 0, tx_bytes=st.tx_bytes if st else 0, 
        ))
    return out 