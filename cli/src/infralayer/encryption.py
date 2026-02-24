import json
import hashlib
import getpass
import platform
from pathlib import Path
from base64 import urlsafe_b64encode, urlsafe_b64decode

from cryptography.fernet import Fernet


def get_encryption_key() -> bytes:
    """Derive encryption key from machine-specific information."""
    hostname = platform.node()
    username = getpass.getuser()
    salt = f"{hostname}:{username}:infralayer"

    key_material = hashlib.sha256(salt.encode()).digest()
    # Fernet requires a 32-byte key, base64-encoded
    return urlsafe_b64encode(key_material)


def encrypt_data(data: dict) -> str:
    """Encrypt a dictionary to a string."""
    key = get_encryption_key()
    fernet = Fernet(key)
    json_bytes = json.dumps(data).encode()
    encrypted = fernet.encrypt(json_bytes)
    return urlsafe_b64encode(encrypted).decode()


def decrypt_data(encrypted: str) -> dict:
    """Decrypt a string back to a dictionary."""
    key = get_encryption_key()
    fernet = Fernet(key)
    encrypted_bytes = urlsafe_b64decode(encrypted.encode())
    decrypted = fernet.decrypt(encrypted_bytes)
    return json.loads(decrypted.decode())


def secure_file_write(path: Path, data: str) -> None:
    """Write data to a file with secure permissions (0600)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(data)
    path.chmod(0o600)


def secure_file_read(path: Path) -> str | None:
    """Read data from a file if it exists."""
    if not path.exists():
        return None
    return path.read_text()
