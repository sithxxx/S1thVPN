from dataclasses import dataclass 
from datetime import UTC, datetime, timedelta 
from ipaddress import  IPv4Address 

from sqlalchemy import select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession 

from ..config_render import render_own_config 
from ..dataplane.base import PeerSpec
from ..ipam import IPAM
from ..keys import generate_keypair
from ..models import AuditEvent, Device, Invite, VpnUser
from ..security import new_token, seal, token_hash, unseal 
from ..settings import Settings

class DeviceLimitReached(Exception):
    pass

@dataclass 
class NewDevice:
    device: Device
    invite_url: str 
    
async def create_device(
    session: AsyncSession, settings: Settings, user_id: int, name: str, actor: str
) -> NewDevice:
    user = await session.get(VpnUser, user_id)
    if user is None or not user.is_active:
        raise LookupError("пользователь не найден или отключён")
    count = len((await session.scalars(select(Device.id).where(Device.user_id == user_id))).all())
    if count >= user.max_devices:
        raise DeviceLimitReached(f"лимит устройств: {user.max_devices}") 
    
    ipam = IPAM(settings.own_subnet)
    kp = generate_keypair()
    for _ in range(5):
        used = {IPv4Address(ip) for ip in (await session.scalars(select(Device.tunnel_ip))).all()}
        ip = ipam.allocate(used)
        device = Device(used_id=user_id, name=name, public_key=kp.public, tunnel_ip=str(ip))
        try:
            async with session.begin_nested():
                session.add(device)
            break
        except IntegrityError:
            continue 
    else:
        raise RuntimeError("не удалось выделить IP")
    
    token = new_token()
    session.add(Invite(
        device_id=device.id,
        token_hash=token_hash(token),
        sealed_config=seal(token, render_own_config(settings, kp.private, ip)),
        expires_at=datetime.now(UTC) + timedelta(hours=settings.invite_ttl_hours),
    ))
    session.add(AuditEvent(actor=actor, action="device.create", details={"device_id": device.id, "ip": str(ip)}))
    await session.commit()
    return NewDevice(device, f"{settings.public_base_url}/i/{token}")

async def redeem_invite(session: AsyncSession, token: str) -> str | None:
    """Отдаёт конфиг ОДИН раз, после чего стирает его"""
    inv = await session.scalar(select(Invite).where(Invite.token_hash == token_hash(token)))
    if inv is None or inv.used_at is not None or inv.sealed_config is None:
        return None 
    config = unseal(token, inv.sealed_config)
    inv.used_at, inv.sealed_config = datetime.now(UTC), None
    await session.commit()
    return config

async def desired_peers(session: AsyncSession) -> list[PeerSpec]:
    """Что ДОЛЖНО быть в собственном датаплейне (вход для reconciler)"""
    rows = await session.execute(
        select(Device.public_key, Device.tunnel_ip, VpnUser.rate_limit_mbps)
        .join(VpnUser)
        .where(Device.is_active, VpnUser.is_active, Device.dataplane == "own")
    )
    return [PeerSpec(pk, ip, mbps) for pk, ip, mbps in rows.all()]