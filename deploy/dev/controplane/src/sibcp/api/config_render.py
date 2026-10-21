from ipaddress import IPv4Address, IPv4Network

from .setting import Settings 

def render_own_config(s: Settings, private_key: str, ip: IPv4Address) -> str:
    """Конфиг для нашего клиента sibtun (TOML)."""
    prefix = IPv4Network(s.own_subnet).prefixlen
    dns = ", ".join(f'"{d}"' for d in s.client_dns)
    return (
        "# SibTunnel client config.  Храните в секрете: внутри приватный ключ.\n"
        "[interface]\n"
        f'private_key = "{private_key}"\n'
        f'address = "{ip}/{prefix}"\n'
        f"dns = [{dns}]\n"
        f"mtu = {s.client_mtu}\n\n"
        "[server]\n"
        f'public_key = "{s.server_public_key}"\n'
        f'endpoint = "{s.server_endpoint}"\n'
        "keeplive_seconds = 25\n"
    )
    