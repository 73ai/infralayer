import os
import shutil
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

CONFIG_DIR = pathlib.Path.home() / ".config" / "infralayer"
CONFIG_FILE = CONFIG_DIR / "config.yaml"


def is_dev_mode() -> bool:
    return os.environ.get("INFRALAYER_DEV_MODE", "").lower() == "true"


def is_yolo_mode() -> bool:
    """Check if yolo mode is enabled (skips permission prompts)."""
    return os.environ.get("INFRALAYER_YOLO_MODE", "").lower() == "true"


def set_yolo_mode(enabled: bool) -> None:
    """Set yolo mode via environment variable."""
    os.environ["INFRALAYER_YOLO_MODE"] = "true" if enabled else "false"


def get_api_base_url() -> str:
    if is_dev_mode():
        return "http://localhost:8080"
    return "https://api.infralayer.dev"


def get_console_base_url() -> str:
    if is_dev_mode():
        return "http://localhost:5173"
    return "https://app.infralayer.dev"


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


def migrate_legacy_config() -> None:
    """Migrate config from ~/.config/infragpt/ to ~/.config/infralayer/ if needed."""
    legacy_dir = pathlib.Path.home() / ".config" / "infragpt"
    if legacy_dir.exists() and not CONFIG_DIR.exists():
        shutil.copytree(legacy_dir, CONFIG_DIR)
        console.print(
            "[yellow]Migrated config from ~/.config/infragpt/ to ~/.config/infralayer/[/yellow]"
        )


def init_config() -> None:
    """Initialize configuration file with environment variables if it doesn't exist."""
    migrate_legacy_config()

    if CONFIG_FILE.exists():
        return

    CONFIG_DIR.mkdir(parents=True, exist_ok=True)

    from infralayer.history import init_history_dir

    init_history_dir()

    config: Dict[str, Any] = {}

    openai_key = os.getenv("OPENAI_API_KEY")
    anthropic_key = os.getenv("ANTHROPIC_API_KEY")
    env_model = os.getenv("INFRALAYER_MODEL")

    if anthropic_key and (not env_model or env_model == "claude"):
        config["model"] = "anthropic:claude-sonnet-4-20250514"
        config["api_key"] = anthropic_key
    elif openai_key and (not env_model or env_model == "gpt4o"):
        config["model"] = "openai:gpt-4o"
        config["api_key"] = openai_key

    if config:
        save_config(config)
