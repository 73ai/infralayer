"""
Shell command execution module for InfraGPT CLI agent.

This module provides functionality for executing shell commands with:
- Real-time streaming output
- TTY support for interactive commands
- Timeout handling (1 minute default)
- ESC key for early termination
- Environment variable persistence across commands
"""

import os
import sys
import signal
import subprocess
import threading
import time
from typing import Optional, Dict, Tuple
import pty
import select

from rich.console import Console

from .container import ExecutorInterface

console = Console()


class CommandExecutor(ExecutorInterface):
    """Handles shell command execution with streaming and timeout."""

    def __init__(self, timeout: int = 60, env: Optional[Dict[str, str]] = None):
        """
        Initialize command executor.

        Args:
            timeout: Command timeout in seconds (default: 60)
            env: Environment variables to persist across commands
        """
        self.timeout = timeout
        self.env = env or os.environ.copy()
        self.current_process = None
        self.cancelled = False
        self.output_buffer = []

    def execute_command(self, command: str) -> Tuple[int, str, bool]:
        """
        Execute a shell command with streaming output and timeout.

        Args:
            command: Shell command to execute

        Returns:
            Tuple of (exit_code, output, was_cancelled)
        """
        self.cancelled = False
        self.output_buffer = []

        console.print(f"[bold cyan]Executing:[/bold cyan] {command}")
        console.print("[dim]Press ESC to cancel command...[/dim]\n")

        try:
            master_fd, slave_fd = pty.openpty()

            # preexec_fn might not work on all platforms
            popen_args = {
                "shell": True,
                "stdin": slave_fd,
                "stdout": slave_fd,
                "stderr": slave_fd,
                "env": self.env,
            }

            if hasattr(os, "setsid"):
                try:
                    popen_args["preexec_fn"] = os.setsid
                except TypeError:
                    pass

            self.current_process = subprocess.Popen(command, **popen_args)
            os.close(slave_fd)

            timer = threading.Timer(self.timeout, self._timeout_handler)
            timer.start()

            esc_thread = threading.Thread(target=self._esc_listener, daemon=True)
            esc_thread.start()

            output = self._stream_output(master_fd)

            exit_code = None
            poll_interval = 0.1  # seconds
            start_time = time.time()
            while True:
                ret = self.current_process.poll()
                if ret is not None:
                    exit_code = ret
                    break
                if self.cancelled:
                    exit_code = -1
                    break
                if (time.time() - start_time) > self.timeout:
                    exit_code = -1
                    break
                time.sleep(poll_interval)

            timer.cancel()
            os.close(master_fd)

            if self.cancelled:
                console.print("\n[bold yellow]Command cancelled by user[/bold yellow]")
                return -1, output, True

            return exit_code, output, False

        except OSError as e:
            console.print(f"[bold red]Error executing command:[/bold red] {e}")
            return -1, str(e), False
        finally:
            self.current_process = None

    def _stream_output(self, fd: int) -> str:
        """Stream output from command in real-time."""
        output_lines = []

        try:
            while True:
                if self.current_process and self.current_process.poll() is not None:
                    try:
                        ready, _, _ = select.select([fd], [], [], 0.1)
                        if ready:
                            data = os.read(fd, 4096).decode("utf-8", errors="replace")
                            if data:
                                output_lines.append(data)
                                console.print(data, end="")
                                console.file.flush()  # Force flush for real-time output
                    except (OSError, ValueError):
                        pass
                    break

                try:
                    ready, _, _ = select.select([fd], [], [], 0.1)
                    if ready:
                        data = os.read(fd, 4096).decode("utf-8", errors="replace")
                        if data:
                            output_lines.append(data)
                            console.print(data, end="")
                            console.file.flush()

                    if self.cancelled:
                        break

                except (OSError, ValueError):
                    break
                except KeyboardInterrupt:
                    self.cancelled = True
                    self._terminate_command()
                    break

        except OSError as e:
            console.print(f"[bold red]Error streaming output:[/bold red] {e}")

        return "".join(output_lines)

    def _timeout_handler(self) -> None:
        """Handle command timeout."""
        if self.current_process and self.current_process.poll() is None:
            console.print(
                f"\n[bold yellow]Command timed out after {self.timeout} seconds[/bold yellow]"
            )
            self._terminate_command()

    def _terminate_command(self) -> None:
        """Terminate the current command."""
        if self.current_process:
            try:
                pgid = os.getpgid(self.current_process.pid)
            except (OSError, ProcessLookupError):
                return
            try:
                os.killpg(pgid, signal.SIGTERM)
                time.sleep(1)
                if self.current_process.poll() is None:
                    os.killpg(pgid, signal.SIGKILL)
            except (OSError, ProcessLookupError):
                pass

    def _esc_listener(self) -> None:
        """Listen for ESC key to cancel command."""
        try:
            import termios
            import tty

            old_settings = termios.tcgetattr(sys.stdin)

            try:
                tty.setcbreak(sys.stdin.fileno())

                while self.current_process and self.current_process.poll() is None:
                    if sys.stdin in select.select([sys.stdin], [], [], 0.1)[0]:
                        key = sys.stdin.read(1)
                        if key == "\x1b":
                            self.cancelled = True
                            self._terminate_command()
                            break
            finally:
                termios.tcsetattr(sys.stdin, termios.TCSADRAIN, old_settings)

        except ImportError:
            pass  # termios not available (Windows)
        except OSError:
            pass  # Terminal I/O error

    def update_environment(self, env_vars: Dict[str, str]) -> None:
        """Update environment variables for future commands."""
        self.env.update(env_vars)

    def get_environment(self) -> Dict[str, str]:
        """Get current environment variables."""
        return self.env.copy()

    def cleanup(self) -> None:
        """No-op cleanup for host executor - implements ExecutorInterface."""
        pass


def parse_environment_changes(output: str) -> Dict[str, str]:
    """
    Parse command output for environment variable changes.

    This is a simple implementation that looks for export statements
    in the output. More sophisticated parsing could be added later.

    Args:
        output: Command output to parse

    Returns:
        Dictionary of environment variable changes
    """
    env_changes = {}
    lines = output.split("\n")
    for line in lines:
        line = line.strip()
        if line.startswith("export ") and "=" in line:
            try:
                export_part = line[7:]
                if "=" in export_part:
                    var, value = export_part.split("=", 1)
                    value = value.strip("\"'")
                    env_changes[var] = value
            except ValueError:
                pass

    return env_changes
