import os
import json
import uuid
import datetime
import pathlib
import re
from typing import List, Dict, Any, Optional

from rich.console import Console

console = Console()

HISTORY_DIR = pathlib.Path.home() / ".config" / "infralayer" / "history"
HISTORY_DB_FILE = HISTORY_DIR / "history.jsonl"

INTERACTION_FIELD_ALLOWLIST = {
    "agent_conversation_v2": ["user_input", "assistant_response", "model", "timestamp"],
}


def sanitize_sensitive_data(data: Any) -> Any:
    """
    Recursively sanitize sensitive data from logs.
    Replaces API keys, passwords, and tokens with masked values.
    """
    if isinstance(data, dict):
        sanitized = {}
        for key, value in data.items():
            key_lower = key.lower()
            sensitive_keywords = [
                "password",
                "api_key",
                "apikey",
                "token",
                "secret",
                "credential",
                "auth",
                "access_key",
                "access_token",
                "private_key",
            ]
            if any(sensitive in key_lower for sensitive in sensitive_keywords):
                continue
            else:
                sanitized[key] = sanitize_sensitive_data(value)
        return sanitized
    elif isinstance(data, list):
        return [sanitize_sensitive_data(item) for item in data]
    elif isinstance(data, str):
        patterns = [
            (r"sk-[A-Za-z0-9]{20,}", "sk-***REDACTED***"),
            (r"Bearer\s+[A-Za-z0-9\-._~+/=]{20,}", "Bearer ***REDACTED***"),
            (r"\bAKIA[0-9A-Z]{16}\b", "AKIA***REDACTED***"),
            (r"\bASIA[0-9A-Z]{16}\b", "ASIA***REDACTED***"),
            (r"\b[A-Za-z0-9/+=]{40}\b", "***REDACTED***"),
            (r"\bghp_[A-Za-z0-9]{36}\b", "ghp_***REDACTED***"),
            (r"\bgithub_pat_[A-Za-z0-9_]{82,}\b", "github_pat_***REDACTED***"),
            (r"\bxox[baprs]-[A-Za-z0-9-]{10,}\b", "xox***REDACTED***"),
            (
                r"-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----",
                "***REDACTED PEM KEY***",
            ),
            (
                r"\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b",
                "***REDACTED JWT***",
            ),
            (r"\b[A-Za-z0-9]{32,}\b", "***REDACTED***"),
        ]
        result = data
        for pattern, replacement in patterns:
            result = re.sub(pattern, replacement, result)
        return result
    else:
        return data


def log_interaction(interaction_type: str, data: Dict[str, Any]):
    """Log user interaction to the history database file."""
    try:
        HISTORY_DIR.mkdir(parents=True, exist_ok=True)
        try:
            HISTORY_DIR.chmod(0o700)
        except Exception:
            pass

        sanitized_data = sanitize_sensitive_data(data)
        allowed_fields = INTERACTION_FIELD_ALLOWLIST.get(interaction_type, [])
        if allowed_fields and isinstance(sanitized_data, dict):
            safe_data = {
                k: sanitized_data[k] for k in allowed_fields if k in sanitized_data
            }
        else:
            safe_data = {}

        entry = {
            "id": str(uuid.uuid4()),
            "timestamp": datetime.datetime.now().isoformat(),
            "type": interaction_type,
            "data": safe_data,
        }

        with open(HISTORY_DB_FILE, "a", encoding="utf-8") as f:
            f.write(json.dumps(entry) + "\n")

        os.chmod(HISTORY_DB_FILE, 0o600)

    except Exception as e:
        if "verbose" in data and data.get("verbose"):
            console.print(f"[dim]Warning: Could not log interaction: {e}[/dim]")


def get_interaction_history(limit: int = 100) -> List[Dict[str, Any]]:
    """Retrieve the most recent interaction history entries."""
    if not HISTORY_DB_FILE.exists():
        return []

    try:
        entries = []
        with open(HISTORY_DB_FILE, "r", encoding="utf-8") as f:
            for line in f:
                if line.strip():
                    entries.append(json.loads(line))

        return list(reversed(entries[-limit:]))
    except Exception as e:
        console.print(f"[yellow]Warning:[/yellow] Could not read history: {e}")
        return []


def init_history_dir():
    """Initialize history directory if it doesn't exist."""
    HISTORY_DIR.mkdir(parents=True, exist_ok=True)
    try:
        HISTORY_DIR.chmod(0o700)
    except Exception:
        pass


def display_history_entry(i: int, entry: Dict[str, Any]):
    """Format and display a history entry."""
    entry_type = entry.get("type", "unknown")
    timestamp = entry.get("timestamp", "")
    timestamp_short = timestamp.split("T")[0] if "T" in timestamp else timestamp

    if entry_type == "command_generation":
        data = entry.get("data", {})
        model = data.get("model", "unknown")
        prompt = data.get("prompt", "")
        result = data.get("result", "")

        console.print(
            f"\n[dim]{i + 1}. {timestamp_short}[/dim] [bold blue]Command Generation[/bold blue] [dim]({model})[/dim]"
        )
        console.print(f"[bold cyan]Prompt:[/bold cyan] {prompt}")
        console.print(f"[bold green]Result:[/bold green] {result}")

    elif entry_type == "command_action":
        data = entry.get("data", {})
        action = data.get("action", "unknown")
        command = data.get("processed_command", "")
        params = data.get("parameters", {})

        console.print(
            f"\n[dim]{i + 1}. {timestamp_short}[/dim] [bold magenta]Command Action[/bold magenta] [dim]({action})[/dim]"
        )
        console.print(f"[bold cyan]Command:[/bold cyan] {command}")
        if params:
            console.print(
                f"[bold yellow]Parameters:[/bold yellow] {json.dumps(params)}"
            )

    elif entry_type == "command_execution":
        data = entry.get("data", {})
        command = data.get("command", "")
        exit_code = data.get("exit_code", -1)
        duration = data.get("duration_ms", 0) / 1000

        console.print(
            f"\n[dim]{i + 1}. {timestamp_short}[/dim] [bold green]Command Execution[/bold green] [dim](exit: {exit_code}, {duration:.2f}s)[/dim]"
        )
        console.print(f"[bold cyan]Command:[/bold cyan] {command}")

    elif entry_type == "agent_conversation":
        data = entry.get("data", {})
        model = data.get("model", "unknown")
        user_input = data.get("user_input", "")
        assistant_response = data.get("assistant_response", "")
        tool_calls = data.get("tool_calls", [])

        console.print(
            f"\n[dim]{i + 1}. {timestamp_short}[/dim] [bold purple]Agent Conversation[/bold purple] [dim]({model})[/dim]"
        )
        console.print(f"[bold cyan]User:[/bold cyan] {user_input}")

        if len(assistant_response) > 200:
            response_preview = assistant_response[:200] + "..."
        else:
            response_preview = assistant_response
        console.print(f"[bold green]Assistant:[/bold green] {response_preview}")

        if tool_calls:
            console.print(
                f"[bold yellow]Tool Calls:[/bold yellow] {len(tool_calls)} executed"
            )

    else:
        console.print(
            f"\n[dim]{i + 1}. {timestamp_short}[/dim] [bold]{entry_type}[/bold]"
        )
        console.print(json.dumps(entry.get("data", {}), indent=2))


def history_command(
    limit: int = 10, type: Optional[str] = None, export: Optional[str] = None
):
    """View or export interaction history."""
    if not HISTORY_DB_FILE.exists():
        console.print("[yellow]No history found.[/yellow]")
        return

    entries = get_interaction_history(limit=limit)

    if not entries:
        console.print("[yellow]No history entries found.[/yellow]")
        return

    if type:
        entries = [entry for entry in entries if entry.get("type") == type]
        if not entries:
            console.print(
                f"[yellow]No history entries found with type '{type}'.[/yellow]"
            )
            return

    if export:
        try:
            with open(export, "w", encoding="utf-8") as f:
                for entry in entries:
                    f.write(json.dumps(entry) + "\n")
            console.print(
                f"[green]Exported {len(entries)} history entries to {export}[/green]"
            )
            return
        except Exception as e:
            console.print(f"[bold red]Error exporting history:[/bold red] {e}")
            return

    console.print(f"[bold]Last {len(entries)} interaction(s):[/bold]")

    for i, entry in enumerate(entries):
        display_history_entry(i, entry)
