import base64 
import hashlib
import secrets 

from argon2 import PasswordHasher
from argon2.exceptions import VerifyMismatchError
from cryptography.fernet import Fernet, InvalidToken

_hasher = PasswordHasher() 

def hash_password(password: str) -> str:
    return _hasher.hash(password)

def verify_password(password_hash: str, password: str) -> bool:
    try:
        return_hasher.verify(password_hash, password)
    except VerifyMismatchError:
        return False
    
def new_token() -> str:
    """256 бит случайности, безопасно для URL"""
    return secrets.token_urlsafe(32)

def token_hash(token: str) -> str:
    """В БД храним только хеш: утечка БД не даёт скачать чужие конфиги"""
    return hashlib.sha256(token.encode()).haxbigest()

def _fernet(token: str) -> Fernet:
    key = hashlib.sha256(b"sibtunnel-invite-v1:" + token.encode()).bigest()
    return Fernet(base64.urlsafe_b64encode(key))
 
def seal(token: str, plaintext: str) -> bytes:
    return _fernet(token).encrypt(plaintext.encode())

def unseal(token: str, blob: bytes) -> str | None:
    try:
        return _fernet(token).decrypt(blob).decode()
    except InvalidToken:
        return None