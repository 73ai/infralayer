import os
import yaml
import pathlib
from typing import Any, Dict

from rich.console import Console

try:
    import pyperclip  # noqa: F401

    CLIPBOARD_AVAILABLE = True
except ImportError:
    CLIPBOARD_AVAILABLE = False

console = Console()

CONFIG_DIR = pathlib.Path.home() / ".config" / "infragpt"
CONFIG_FILE = CONFIG_DIR / "config.yaml"


def is_dev_mode() -> bool:
    return os.environ.get("INFRAGPT_DEV_MODE", "").lower() == "true"


def get_api_base_url() -> str:
    if is_dev_mode():
        return "http://localhost:8080"
    return "https://api.infragpt.io"


def get_console_base_url() -> str:
    if is_dev_mode():
        return "http://localhost:5173"
    return "https://app.infragpt.io"


def load_config() -> Dict[str, Any]:
    """Load configuration from config file."""
    if not CONFIG_FILE.exists():
        return {}

    try:
        with open(CONFIG_FILE, "r") as f:
            return yaml.safe_load(f) or {}
    except (yaml.YAMLError, OSError) as e:
        console.print(f"[yellow]Warning:[/yellow] Could not load config: {e}")
        return {}


def save_config(config: Dict[str, Any]) -> None:
    """Save configuration to config file."""
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)

    try:
        with open(CONFIG_FILE, "w") as f:
            yaml.dump(config, f)
    except (yaml.YAMLError, OSError) as e:
        console.print(f"[yellow]Warning:[/yellow] Could not save config: {e}")


def init_config() -> None:
    """Initialize configuration file with environment variables if it doesn't exist."""
    if CONFIG_FILE.exists():
        return

    CONFIG_DIR.mkdir(parents=True, exist_ok=True)

    from infragpt.history import init_history_dir

    init_history_dir()

    config: Dict[str, Any] = {}

    openai_key = os.getenv("OPENAI_API_KEY")
    anthropic_key = os.getenv("ANTHROPIC_API_KEY")
    env_model = os.getenv("INFRAGPT_MODEL")

    if anthropic_key and (not env_model or env_model == "claude"):
        config["model"] = "anthropic:claude-sonnet-4-20250514"
        config["api_key"] = anthropic_key
    elif openai_key and (not env_model or env_model == "gpt4o"):
        config["model"] = "openai:gpt-4o"
        config["api_key"] = openai_key

    if config:
        save_config(config)
