from unittest.mock import patch
import pytest

from infragpt.api_client import InfraGPTClient, InfraGPTAPIError


class TestValidateToken:
    def test_returns_true_on_success(self):
        with patch("infragpt.api_client.get_api_base_url", return_value="http://test"):
            client = InfraGPTClient()
        with patch.object(client, "_make_request") as mock_request:
            mock_request.return_value = {"service_account_json": "{}"}
            result = client.validate_token("valid-token")
            assert result is True
            mock_request.assert_called_once_with(
                "POST",
                "/device/credentials/gcp",
                headers={"Authorization": "Bearer valid-token"},
            )

    def test_returns_true_on_404(self):
        with patch("infragpt.api_client.get_api_base_url", return_value="http://test"):
            client = InfraGPTClient()
        with patch.object(client, "_make_request") as mock_request:
            mock_request.side_effect = InfraGPTAPIError(404, "Not found")
            result = client.validate_token("valid-token")
            assert result is True

    def test_raises_on_401(self):
        with patch("infragpt.api_client.get_api_base_url", return_value="http://test"):
            client = InfraGPTClient()
        with patch.object(client, "_make_request") as mock_request:
            mock_request.side_effect = InfraGPTAPIError(401, "Unauthorized")
            with pytest.raises(InfraGPTAPIError) as exc_info:
                client.validate_token("invalid-token")
            assert exc_info.value.status_code == 401

    def test_raises_on_other_errors(self):
        with patch("infragpt.api_client.get_api_base_url", return_value="http://test"):
            client = InfraGPTClient()
        with patch.object(client, "_make_request") as mock_request:
            mock_request.side_effect = InfraGPTAPIError(500, "Server error")
            with pytest.raises(InfraGPTAPIError) as exc_info:
                client.validate_token("token")
            assert exc_info.value.status_code == 500
