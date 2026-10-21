from pydantic import SecretStr
from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="SIBCP_", env_file=".env")
    
    database_url: str = "postresql+asyncpg://sibcp:sibcp@localhost:5432/sibcp"
    secret_key: SecretStr
    public_base_url: str = "https://vpn.example.org"
    cookie_secure: bool = True
    
    uapi_socket: str = "/run/sintunnel/uapi.sock"
    own_subnet: str = "10.77.0.0/22"
    server_endpoint: str = "203.0.113.10:51900"
    server_public_key: str = ""
    
    wg_enabled: bool = False
    wg_interface: str = "wg0"
    wg_subnet: str = "10.78.0.0/22"
    wg_endpoint: str = "203.0.113.10:51820"
    wg_server_public_key: str = ""
    
    client_dns: list[str] = ["1.1.1.1", "9.9.9.9"]
    client_mtu: int = 1420
    default_rate_limit_mbps: int = 50
    invite_ttl_hours: int = 72
    reconcile_interval_s: float = 10.0
    
                                      