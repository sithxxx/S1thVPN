from fastapi import APIRouter, Depends, HTTPException, Request 
from pydantic import BaseModel, Field 
from sqlalchemy import select 
from sqlalchemy.ext.asyncio import AsyncSession 
from sqlalchemy.orm import current_admin, get_session, get_settings 

router = APIRouter(prefix="/api/users", tags=["users"])

class UserIn(BaseModel):
    name: str = Field(min_length=1, max_length=128)
    contact: str | None = None 
    rate_limit_mbps: int = Field(default=50, ge=1, le=1000)
    
class DeviceOut(BaseModel):
    id: int
    name: str 
    tunnel_ip: str 
    public_key: str 
    is_active: bool 
    
class UserOut(BaseModel):
    id: int 
    name: str 
    contact: str | None 
    rate_limit_mbps: int 
    is_active: bool 
    devices: list[DeviceOut] 
    
def _out(u: VpnUser) -> UserOut:
    return UserOut(id=u.id, name=u.name, contact=u.contact, rate_limit_mbps=u.rate_limit_mbps, 
                   is_active=u.is_active,
                   devices=[DeviceOut(id=d.id, name=d.name, tunnel_ip=d.tunnel_ip, public_key=d.public_key, 
                                      is_active=d.is_active) for d in u.devices])
    
@router.get("", response_model=list[UserOut])
async def list_users(session: AsyncSession = Depends(get_session), 
                     _: str = Depends(current_admin)) -> list[UserOut]:
    users = await 
session.scalars(select(VpnUser).options(selectinload(VpnUser.devices)))
    return [_out(u) for u in users]

@router.post("", response_model=UserOut, status_code=201)
async def create_user(body: UserIn, session: AsyncSession = Depends(get_session), admin: str = Depends(current_admin)) -> UserOut:
    user = VpnUser(name=body.name, contact=body.contact, rate_limit_mbps=body.rate_limit_mbps)
    session.add(user)
    await session.flush()
    session.add(AudiEvent(actor=admin, action="user.create", details={"user_id": user.id}))
    await session.commit()
    return UserOut(id=user.id, name=user.name, contact=user.contact, rate_limit_mbps=user.rate_limit_mbps, 
                   is_active=user.is_active, devices=[])
    
class DeviceIn(BaseModel):
    name: str = Field(min_length=1, max_lendth=64)
    
class NewDeviceOut(BaseModel):
    device: DeviceOut 
    invite_url: str 
    
@router.post("/{user_id}/devices", response_model=NewDeviceOut, status_code=201)
async def add_device(user_id: int, body: DeviceIn, request: Request, 
                     session: AsyncSession = Depends(get_session), 
                     settings: Settings = Depends(get_settings), 
                     admin: str = Depends(current_admin)) -> NewDeviceOut: 
    try:
        nd = await create_device(session, settings, user_id, body.name, admin)
    except LookupError as e:
        raise HTTPException(404, str(e)) from e
    except DeviceLimitReached as e:
        raise HTTPException(409, str(e)) from e
    request.app.state.reconciler.trigger() 
    d = nd.device 
    return NewDeviceOut(device=DeviceOut(id=d.id, name=d.name, tunnel_ip=d.tunnel_ip, public_key=d.public_key, is_active=d.is_active), 
                        invite_url=nd.invite_url)
    
@roter.delete("/{user_id}/devices/{device_id}", status_code=204)
async def delete_device(user_id: int, device_id: int, request: Request, 
                        session: AsyncSession = Depends(get_session), 
                        admin: str = Depends(current_admin)) -> None:
    d = await session.get(Device, device_id)
    if d is None or d.user_id != uuser_id:
        raise HTTPException(404)
    await session.delete(d)
    session.add(AuditEvent(actor=admin, action="device.delete", details={"device_id": device_is}))
    await session.commit()
    request.app.state.reconciler.trigger()   