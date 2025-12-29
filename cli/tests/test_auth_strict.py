from unittest.mock import MagicMock, patch
import pytest

from infragpt.auth import (
    validate_token_with_api,
    refresh_token_strict,
    fetch_gcp_credentials_strict,
    fetch_gke_cluster_info_strict,
    AuthStatus,
)
from infragpt.api_client import (
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


class TestValidateTokenWithApi:
    def test_raises_when_not_authenticated(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(authenticated=False)
            with pytest.raises(AuthValidationError, match="Not authenticated"):
                validate_token_with_api()

    def test_raises_on_401(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.validate_token.side_effect = InfraGPTAPIError(
                    401, "Unauthorized"
                )
                mock_client_class.return_value = mock_client
                with pytest.raises(
                    AuthValidationError, match="Token is invalid or expired"
                ):
                    validate_token_with_api()

    def test_raises_on_connection_error(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                import httpx

                mock_client.validate_token.side_effect = httpx.ConnectError(
                    "Connection failed"
                )
                mock_client_class.return_value = mock_client
                with pytest.raises(
                    AuthValidationError, match="Failed to connect to server"
                ):
                    validate_token_with_api()

    def test_success(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.validate_token.return_value = True
                mock_client_class.return_value = mock_client
                validate_token_with_api()  # Should not raise


class TestRefreshTokenStrict:
    def test_raises_when_no_auth_data(self):
        with patch("infragpt.auth._load_auth_data") as mock_load:
            mock_load.return_value = None
            with pytest.raises(TokenRefreshError, match="No auth data found"):
                refresh_token_strict()

    def test_raises_when_no_refresh_token(self):
        with patch("infragpt.auth._load_auth_data") as mock_load:
            mock_load.return_value = {"access_token": "token"}
            with pytest.raises(TokenRefreshError, match="No refresh token available"):
                refresh_token_strict()

    def test_raises_on_api_error(self):
        with patch("infragpt.auth._load_auth_data") as mock_load:
            mock_load.return_value = {
                "access_token": "token",
                "refresh_token": "refresh",
            }
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.refresh_token.side_effect = InfraGPTAPIError(
                    400, "Invalid refresh token"
                )
                mock_client_class.return_value = mock_client
                with pytest.raises(TokenRefreshError, match="Failed to refresh token"):
                    refresh_token_strict()


class TestFetchGcpCredentialsStrict:
    def test_raises_when_not_authenticated(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(authenticated=False)
            with pytest.raises(GCPCredentialError, match="Not authenticated"):
                fetch_gcp_credentials_strict()

    def test_raises_on_404(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.get_gcp_credentials.side_effect = InfraGPTAPIError(
                    404, "Not found"
                )
                mock_client_class.return_value = mock_client
                with pytest.raises(
                    GCPCredentialError,
                    match="No GCP credentials configured for your organization",
                ):
                    fetch_gcp_credentials_strict()

    def test_success(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.get_gcp_credentials.return_value = GCPCredentials(
                    service_account_json='{"type":"service_account"}',
                    project_id="test-project",
                )
                mock_client_class.return_value = mock_client
                result = fetch_gcp_credentials_strict()
                assert result.project_id == "test-project"


class TestFetchGkeClusterInfoStrict:
    def test_raises_when_not_authenticated(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(authenticated=False)
            with pytest.raises(GKEClusterError, match="Not authenticated"):
                fetch_gke_cluster_info_strict()

    def test_raises_on_404(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.get_gke_cluster_info.side_effect = InfraGPTAPIError(
                    404, "Not found"
                )
                mock_client_class.return_value = mock_client
                with pytest.raises(
                    GKEClusterError,
                    match="No GKE cluster configured for your organization",
                ):
                    fetch_gke_cluster_info_strict()

    def test_success(self):
        with patch("infragpt.auth.get_auth_status") as mock_status:
            mock_status.return_value = AuthStatus(
                authenticated=True, access_token="token"
            )
            with patch("infragpt.auth.InfraGPTClient") as mock_client_class:
                mock_client = MagicMock()
                mock_client.get_gke_cluster_info.return_value = GKEClusterInfo(
                    cluster_name="test-cluster",
                    project_id="test-project",
                    zone="us-central1-a",
                )
                mock_client_class.return_value = mock_client
                result = fetch_gke_cluster_info_strict()
                assert result.cluster_name == "test-cluster"
