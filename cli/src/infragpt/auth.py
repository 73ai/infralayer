import json
import time
import webbrowser
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import httpx
from cryptography.fernet import InvalidToken

from infragpt.config import CONFIG_DIR, console, get_console_base_url
from infragpt.encryption import (
    encrypt_data,
    decrypt_data,
    secure_file_write,
    secure_file_read,
)
from infragpt.api_client import (
    InfraGPTClient,
    InfraGPTAPIError,
    GCPCredentials,
    GKEClusterInfo,
)
from infragpt.exceptions import (
    AuthValidationError,
    TokenRefreshError,
    GCPCredentialError,
    GKEClusterError,
)


AUTH_FILE = CONFIG_DIR / "auth.json"
GCP_CREDENTIALS_FILE = CONFIG_DIR / "gcp_credentials.json"
TOKEN_REFRESH_THRESHOLD_HOURS = 1


@dataclass
class AuthStatus:
    authenticated: bool
    organization_id: Optional[str] = None
    user_id: Optional[str] = None
    access_token: Optional[str] = None
    refresh_token: Optional[str] = None
    expires_at: Optional[str] = None


def _load_auth_data() -> Optional[dict]:
    """Load and decrypt auth data from file."""
    encrypted = secure_file_read(AUTH_FILE)
    if not encrypted:
        return None
    try:
        return decrypt_data(encrypted)
    except (InvalidToken, json.JSONDecodeError):
        return None


def _save_auth_data(data: dict) -> None:
    """Encrypt and save auth data to file."""
    encrypted = encrypt_data(data)
    secure_file_write(AUTH_FILE, encrypted)


def is_authenticated() -> bool:
    """Check if user is authenticated."""
    data = _load_auth_data()
    if not data:
        return False

    expires_at = data.get("expires_at")
    if expires_at:
        try:
            expires_dt = datetime.fromisoformat(expires_at.replace("Z", "+00:00"))
            if expires_dt < datetime.now(timezone.utc):
                return False
        except ValueError:
            pass

    return bool(data.get("access_token"))


def get_auth_status() -> AuthStatus:
    """Get current authentication status."""
    data = _load_auth_data()
    if not data:
        return AuthStatus(authenticated=False)

    return AuthStatus(
        authenticated=is_authenticated(),
        organization_id=data.get("organization_id"),
        user_id=data.get("user_id"),
        access_token=data.get("access_token"),
        refresh_token=data.get("refresh_token"),
        expires_at=data.get("expires_at"),
    )


def get_cli_token() -> Optional[str]:
    """Get CLI access token if available and not expired."""
    status = get_auth_status()
    if not status.authenticated:
        return None
    return status.access_token


def refresh_token_if_needed() -> bool:
    """Refresh token if it's expired or about to expire. Returns True if successful."""
    data = _load_auth_data()
    if not data:
        return False

    expires_at = data.get("expires_at")
    refresh_token = data.get("refresh_token")

    if not refresh_token:
        return False

    # Check if token is expired or expires in less than 1 hour
    should_refresh = False
    if expires_at:
        try:
            expires_dt = datetime.fromisoformat(expires_at.replace("Z", "+00:00"))
            hours_until_expiry = (
                expires_dt - datetime.now(timezone.utc)
            ).total_seconds() / 3600
            should_refresh = hours_until_expiry < TOKEN_REFRESH_THRESHOLD_HOURS
        except ValueError:
            should_refresh = True
    else:
        should_refresh = True

    if not should_refresh:
        return True

    try:
        client = InfraGPTClient()
        result = client.refresh_token(refresh_token)

        from datetime import timedelta

        expires_at_new = datetime.now(timezone.utc).replace(microsecond=0)
        expires_at_new = expires_at_new.replace(tzinfo=None)
        expires_at_new = expires_at_new + timedelta(seconds=result.expires_in)
        expires_at_new = expires_at_new.replace(tzinfo=timezone.utc)

        data["access_token"] = result.access_token
        data["refresh_token"] = result.refresh_token
        data["expires_at"] = expires_at_new.isoformat()

        _save_auth_data(data)
        return True
    except (InfraGPTAPIError, httpx.RequestError):
        return False


def login() -> None:
    """Authenticate with InfraGPT platform using device flow."""
    client = InfraGPTClient()

    console.print("\n[bold]Authenticating with InfraGPT...[/bold]\n")

    try:
        flow = client.initiate_device_flow()
    except InfraGPTAPIError as e:
        console.print(f"[red]Error initiating authentication: {e.message}[/red]")
        raise SystemExit(1)

    # Construct verification URL
    if flow.verification_url.startswith("http"):
        verification_url = flow.verification_url
    else:
        console_url = get_console_base_url()
        path = flow.verification_url.lstrip("/")
        verification_url = f"{console_url}/{path}"

    console.print(f"Visit: [cyan]{verification_url}[/cyan]")
    console.print(f"Enter this code: [bold yellow]{flow.user_code}[/bold yellow]\n")

    try:
        webbrowser.open(verification_url)
        console.print("[dim]Browser opened automatically.[/dim]\n")
    except OSError:
        pass

    console.print("[dim]Waiting for authorization...[/dim]", end="")

    # Poll for authorization
    start_time = time.time()
    poll_interval = flow.interval

    while time.time() - start_time < flow.expires_in:
        time.sleep(poll_interval)

        try:
            result = client.poll_device_flow(flow.device_code)
        except InfraGPTAPIError as e:
            if e.status_code == 410:  # Expired
                console.print("\n[red]Code expired. Please try again.[/red]")
                raise SystemExit(1)
            console.print(f"\n[red]Error: {e.message}[/red]")
            raise SystemExit(1)

        if result.authorized and result.access_token:
            console.print(" [green]authorized![/green]\n")

            # Calculate expiration time
            from datetime import timedelta

            expires_at = datetime.now(timezone.utc) + timedelta(
                seconds=result.expires_in or 604800
            )

            # Save auth data
            auth_data = {
                "access_token": result.access_token,
                "refresh_token": result.refresh_token,
                "organization_id": result.organization_id,
                "user_id": result.user_id,
                "expires_at": expires_at.isoformat(),
            }
            _save_auth_data(auth_data)

            console.print("[green]Successfully logged in![/green]")
            if result.organization_id:
                console.print(f"[dim]Organization ID: {result.organization_id}[/dim]")
            return

        if result.error == "authorization_pending":
            console.print(".", end="", style="dim")
            continue

        console.print(f"\n[red]Error: {result.error}[/red]")
        raise SystemExit(1)

    console.print("\n[red]Authorization timed out. Please try again.[/red]")
    raise SystemExit(1)


def logout() -> None:
    """Remove stored credentials and revoke token."""
    data = _load_auth_data()

    if data and data.get("access_token"):
        try:
            client = InfraGPTClient()
            client.revoke_token(data["access_token"])
        except (InfraGPTAPIError, httpx.RequestError):
            pass  # Token may already be invalid; don't fail logout

    if AUTH_FILE.exists():
        AUTH_FILE.unlink()

    cleanup_credentials()

    console.print("[green]Logged out successfully.[/green]")


def fetch_gcp_credentials() -> Optional[GCPCredentials]:
    """Fetch GCP credentials from server."""
    status = get_auth_status()
    if not status.authenticated or not status.access_token:
        return None

    try:
        client = InfraGPTClient()
        return client.get_gcp_credentials(status.access_token)
    except InfraGPTAPIError:
        return None


def fetch_gke_cluster_info() -> Optional[GKEClusterInfo]:
    """Fetch GKE cluster info from server."""
    status = get_auth_status()
    if not status.authenticated or not status.access_token:
        return None

    try:
        client = InfraGPTClient()
        return client.get_gke_cluster_info(status.access_token)
    except InfraGPTAPIError:
        return None


def write_gcp_credentials_file(credentials: GCPCredentials) -> Optional[Path]:
    """Write GCP credentials to a temporary file. Returns path to file."""
    if not credentials.service_account_json:
        return None

    secure_file_write(GCP_CREDENTIALS_FILE, credentials.service_account_json)
    return GCP_CREDENTIALS_FILE


def cleanup_credentials() -> None:
    """Remove temporary credential files."""
    if GCP_CREDENTIALS_FILE.exists():
        GCP_CREDENTIALS_FILE.unlink()


def validate_token_with_api() -> None:
    """Validate token with backend API. Raises AuthValidationError on failure."""
    status = get_auth_status()
    if not status.authenticated or not status.access_token:
        raise AuthValidationError("Not authenticated")

    try:
        client = InfraGPTClient()
        client.validate_token(status.access_token)
    except InfraGPTAPIError as e:
        if e.status_code == 401:
            raise AuthValidationError("Token is invalid or expired") from e
        raise AuthValidationError(f"Failed to validate token: {e.message}") from e
    except httpx.RequestError as e:
        raise AuthValidationError(f"Failed to connect to server: {e}") from e


def refresh_token_strict() -> None:
    """Refresh token if needed. Raises TokenRefreshError on failure."""
    data = _load_auth_data()
    if not data:
        raise TokenRefreshError("No auth data found")

    expires_at = data.get("expires_at")
    refresh_token = data.get("refresh_token")

    if not refresh_token:
        raise TokenRefreshError("No refresh token available")

    should_refresh = False
    if expires_at:
        try:
            expires_dt = datetime.fromisoformat(expires_at.replace("Z", "+00:00"))
            hours_until_expiry = (
                expires_dt - datetime.now(timezone.utc)
            ).total_seconds() / 3600
            should_refresh = hours_until_expiry < TOKEN_REFRESH_THRESHOLD_HOURS
        except ValueError:
            should_refresh = True
    else:
        should_refresh = True

    if not should_refresh:
        return

    try:
        client = InfraGPTClient()
        result = client.refresh_token(refresh_token)

        from datetime import timedelta

        expires_at_new = datetime.now(timezone.utc).replace(microsecond=0)
        expires_at_new = expires_at_new.replace(tzinfo=None)
        expires_at_new = expires_at_new + timedelta(seconds=result.expires_in)
        expires_at_new = expires_at_new.replace(tzinfo=timezone.utc)

        data["access_token"] = result.access_token
        data["refresh_token"] = result.refresh_token
        data["expires_at"] = expires_at_new.isoformat()

        _save_auth_data(data)
    except InfraGPTAPIError as e:
        raise TokenRefreshError(f"Failed to refresh token: {e.message}") from e
    except httpx.RequestError as e:
        raise TokenRefreshError(f"Failed to connect to server: {e}") from e


def fetch_gcp_credentials_strict() -> GCPCredentials:
    """Fetch GCP credentials from server. Raises GCPCredentialError on failure."""
    status = get_auth_status()
    if not status.authenticated or not status.access_token:
        raise GCPCredentialError("Not authenticated")

    try:
        client = InfraGPTClient()
        return client.get_gcp_credentials(status.access_token)
    except InfraGPTAPIError as e:
        if e.status_code == 404:
            raise GCPCredentialError(
                "No GCP credentials configured for your organization"
            ) from e
        raise GCPCredentialError(f"Failed to fetch GCP credentials: {e.message}") from e
    except httpx.RequestError as e:
        raise GCPCredentialError(f"Failed to connect to server: {e}") from e


def fetch_gke_cluster_info_strict() -> GKEClusterInfo:
    """Fetch GKE cluster info from server. Raises GKEClusterError on failure."""
    status = get_auth_status()
    if not status.authenticated or not status.access_token:
        raise GKEClusterError("Not authenticated")

    try:
        client = InfraGPTClient()
        return client.get_gke_cluster_info(status.access_token)
    except InfraGPTAPIError as e:
        if e.status_code == 404:
            raise GKEClusterError(
                "No GKE cluster configured for your organization"
            ) from e
        raise GKEClusterError(f"Failed to fetch GKE cluster info: {e.message}") from e
    except httpx.RequestError as e:
        raise GKEClusterError(f"Failed to connect to server: {e}") from e
