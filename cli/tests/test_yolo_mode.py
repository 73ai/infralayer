import os
from unittest.mock import patch
import pytest

from infralayer.config import is_yolo_mode, set_yolo_mode
from infralayer.tools import require_permission, ToolExecutionCancelled


class TestIsYoloMode:
    def test_returns_false_by_default(self):
        with patch.dict(os.environ, {}, clear=True):
            assert is_yolo_mode() is False

    def test_returns_false_when_empty(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": ""}):
            assert is_yolo_mode() is False

    def test_returns_true_when_set_true(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "true"}):
            assert is_yolo_mode() is True

    def test_returns_true_case_insensitive(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "TRUE"}):
            assert is_yolo_mode() is True

    def test_returns_true_mixed_case(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "True"}):
            assert is_yolo_mode() is True

    def test_returns_false_for_other_values(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "yes"}):
            assert is_yolo_mode() is False

    def test_returns_false_for_1(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "1"}):
            assert is_yolo_mode() is False


class TestSetYoloMode:
    def test_sets_env_var_true(self):
        with patch.dict(os.environ, {}, clear=True):
            set_yolo_mode(True)
            assert os.environ.get("INFRALAYER_YOLO_MODE") == "true"

    def test_sets_env_var_false(self):
        with patch.dict(os.environ, {"INFRALAYER_YOLO_MODE": "true"}):
            set_yolo_mode(False)
            assert os.environ.get("INFRALAYER_YOLO_MODE") == "false"


class TestRequirePermission:
    def test_skips_prompt_in_yolo_mode(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=True):
            with patch("infralayer.tools.console") as mock_console:
                require_permission("Test action")
                mock_console.print.assert_called_once()
                assert "yolo mode" in str(mock_console.print.call_args)

    def test_prompts_when_not_yolo_mode_and_approved(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", return_value="y"):
                    require_permission("Test action")  # Should not raise

    def test_prompts_when_not_yolo_mode_and_empty_approved(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", return_value=""):
                    require_permission("Test action")  # Should not raise

    def test_prompts_when_not_yolo_mode_and_yes_approved(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", return_value="yes"):
                    require_permission("Test action")  # Should not raise

    def test_raises_on_decline(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", return_value="n"):
                    with pytest.raises(ToolExecutionCancelled):
                        require_permission("Test action")

    def test_raises_on_keyboard_interrupt(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", side_effect=KeyboardInterrupt):
                    with pytest.raises(ToolExecutionCancelled):
                        require_permission("Test action")

    def test_raises_on_eof(self):
        with patch("infralayer.tools.is_yolo_mode", return_value=False):
            with patch("infralayer.tools.console"):
                with patch("builtins.input", side_effect=EOFError):
                    with pytest.raises(ToolExecutionCancelled):
                        require_permission("Test action")
