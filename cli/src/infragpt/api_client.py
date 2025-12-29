from dataclasses import dataclass
from typing import Optional

import httpx

from infragpt.config import get_api_base_url


@dataclass
class DeviceFlowResponse:
    device_code: str
    user_code: str
    verification_url: str
    expires_in: int
    interval: int


@dataclass
class PollResponse:
    authorized: bool
    access_token: Optional[str] = None
    refresh_token: Optional[str] = None
    expires_in: Optional[int] = None
    organization_id: Optional[str] = None
    user_id: Optional[str] = None
    error: Optional[str] = None


@dataclass
class TokenResponse:
    access_token: str
    refresh_token: str
    expires_in: int


@dataclass
class GCPCredentials:
    service_account_json: str
    project_id: Optional[str] = None


@dataclass
class GKEClusterInfo:
    cluster_name: str
    project_id: str
    zone: Optional[str] = None
    region: Optional[str] = None


class InfraGPTAPIError(Exception):
    def __init__(self, status_code: int, message: str):
        self.status_code = status_code
        self.message = message
        super().__init__(f"API Error ({status_code}): {message}")


class InfraGPTClient:
    def __init__(self, timeout: float = 30.0):
        self.api_base_url = get_api_base_url()
        self.timeout = timeout

    def _make_request(
        self,
        method: str,
        endpoint: str,
        json_data: Optional[dict] = None,
        headers: Optional[dict] = None,
    ) -> dict:
        url = f"{self.api_base_url}{endpoint}"
        request_headers = {"Content-Type": "application/json"}
        if headers:
            request_headers.update(headers)

        try:
            with httpx.Client(timeout=self.timeout) as client:
                response = client.request(
                    method,
                    url,
                    json=json_data,
                    headers=request_headers,
                )

                if response.status_code >= 400:
                    try:
                        error_data = response.json()
                        message = (
                            error_data.get("error")
                            or error_data.get("message")
                            or response.text
                        )
                    except Exception:
                        message = response.text or f"HTTP {response.status_code}"
                    raise InfraGPTAPIError(response.status_code, message)

                return response.json()
        except httpx.TimeoutException:
            raise InfraGPTAPIError(0, "Request timed out")
        except httpx.ConnectError:
            raise InfraGPTAPIError(
                0, f"Could not connect to server: {self.api_base_url}"
            )

    def initiate_device_flow(self) -> DeviceFlowResponse:
        data = self._make_request("POST", "/device/auth/initiate")
        return DeviceFlowResponse(
            device_code=data["device_code"],
            user_code=data["user_code"],
            verification_url=data["verification_url"],
            expires_in=data["expires_in"],
            interval=data["interval"],
        )

    def poll_device_flow(self, device_code: str) -> PollResponse:
        data = self._make_request(
            "POST",
            "/device/auth/poll",
            json_data={"device_code": device_code},
        )
        return PollResponse(
            authorized=data.get("authorized", False),
            access_token=data.get("access_token"),
            refresh_token=data.get("refresh_token"),
            expires_in=data.get("expires_in"),
            organization_id=data.get("organization_id"),
            user_id=data.get("user_id"),
            error=data.get("error"),
        )

    def refresh_token(self, refresh_token: str) -> TokenResponse:
        data = self._make_request(
            "POST",
            "/device/auth/refresh",
            json_data={"refresh_token": refresh_token},
        )
        return TokenResponse(
            access_token=data["access_token"],
            refresh_token=data["refresh_token"],
            expires_in=data["expires_in"],
        )

    def revoke_token(self, access_token: str) -> bool:
        self._make_request(
            "POST",
            "/device/auth/revoke",
            headers={"Authorization": f"Bearer {access_token}"},
        )
        return True

    def get_gcp_credentials(self, access_token: str) -> GCPCredentials:
        data = self._make_request(
            "POST",
            "/device/credentials/gcp",
            headers={"Authorization": f"Bearer {access_token}"},
        )
        return GCPCredentials(
            service_account_json=data["service_account_json"],
            project_id=data.get("project_id"),
        )

    def get_gke_cluster_info(self, access_token: str) -> GKEClusterInfo:
        data = self._make_request(
            "POST",
            "/device/credentials/gke",
            headers={"Authorization": f"Bearer {access_token}"},
        )
        return GKEClusterInfo(
            cluster_name=data["cluster_name"],
            project_id=data["project_id"],
            zone=data.get("zone"),
            region=data.get("region"),
        )

    def validate_token(self, access_token: str) -> bool:
        """Validate token with backend API. Returns True if valid, raises on 401."""
        try:
            self._make_request(
                "POST",
                "/device/credentials/gcp",
                headers={"Authorization": f"Bearer {access_token}"},
            )
            return True
        except InfraGPTAPIError as e:
            if e.status_code == 404:
                return True  # Valid token but no GCP credentials configured
            raise
