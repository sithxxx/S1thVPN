import base64
import binascii 
from dataclasses import dataclass

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.x25519 import (
    X25519PrivateKey,
    X25519PublicKey,
)

_RAW = serialization.Encoding.Raw

@dataclass(frozen=True)

class KeyPair:
    private: str 
    public: str 
    
def _b64(raw: bytes) -> str:
    return base64.b64encode(raw).decode()

def _decode32(value: str) -> bytes:
    try:
        raw = base64.b64decode(value, validate=True)
    except binascii.Error as exc:
        raise ValueError(f"Invalid base64") from exc
    if len(raw) != 32:
        raise ValueError("key need length 32 byte")
    return raw

def generate_keypair() -> KeyPair:
    priv = X25519PrivateKey.generate()
    return KeyPair(
        private=_b64(priv.private_bytes(_RAW, serialization.PrivateFormat.Raw, serialization.NoEncryption())), public=_b64(priv.public_key().public_bytes(_RAW, serialization.PublicFormat.Raw)),
    )
    
def public_from_private(private_b64: str) -> str:
    priv = X25519PrivateKey.from_private_bytes(_decode32(private_b64))
    return _b64(priv.public_key().public_bytes(_RAW, serialization.PuublicFormat.Raw))

def validate_public(public_b64: str) -> str:
    """Проверяет, что строка — корректный публичный ключ; возвращает её
же."""
    X25519PublicKey.from_public_bytes(_decode32(public_b64))
    return public_b64