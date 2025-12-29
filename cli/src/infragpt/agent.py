"""
InfraGPT Shell Agent for interactive command execution.
"""

from typing import List, Dict, Any, Optional
from datetime import datetime

from rich.console import Console
from rich.panel import Panel

from .llm_adapter import get_llm_adapter
from .history import log_interaction
from .tools import ToolExecutionCancelled

import pathlib
from prompt_toolkit import PromptSession
from prompt_toolkit.history import FileHistory


def get_system_prompt():
    return """You are an intelligent shell operations assistant. You help users with:

1. Infrastructure and system administration tasks
2. Debugging and troubleshooting issues  
3. Running shell commands safely with user confirmation
4. Analyzing system logs and performance

You have access to shell command execution tools. Always:
- Ask for confirmation before running commands
- Explain what commands do before executing them
- Be cautious with destructive operations
- Provide helpful context and suggestions

Be concise but thorough in your responses."""


console = Console()


class ConversationContext:
    """Manages conversation context with message history."""

    def __init__(self, max_messages: int = 5):
        """Initialize conversation context."""
        self.max_messages = max_messages
        self.messages: List[Dict[str, Any]] = []
        self.system_message = None

    def add_message(
        self,
        role: str,
        content: str,
        tool_calls: Optional[List[Dict[str, Any]]] = None,
        tool_call_id: Optional[str] = None,
    ) -> None:
        """Add a message to the conversation context."""
        message_dict: Dict[str, Any] = {"role": role, "content": content}

        if tool_calls:
            message_dict["tool_calls"] = tool_calls

        if tool_call_id:
            message_dict["tool_call_id"] = tool_call_id
            message_dict["name"] = (
                "execute_shell_command"  # Add tool name for compatibility
            )

        if role == "system":
            self.system_message = message_dict
        else:
            self.messages.append(message_dict)
            if len(self.messages) > self.max_messages:
                self.messages = self.messages[-self.max_messages :]

    def get_context_messages(self) -> List[Dict[str, Any]]:
        """Get messages formatted for LLM API."""
        context = []
        if self.system_message:
            context.append(self.system_message)
        context.extend(self.messages)

        return context

    def clear(self):
        """Clear conversation context."""
        self.messages = []


class ModernShellAgent:
    """Modern shell agent using direct SDK integration."""

    def __init__(self, model_string: str, api_key: str, verbose: bool = False):
        """Initialize shell agent."""
        self.model_string = model_string
        self.api_key = api_key
        self.verbose = verbose
        self.context = ConversationContext()

        self.llm_adapter = get_llm_adapter(
            model_string=model_string, api_key=api_key, verbose=verbose
        )

        self._setup_command_history()
        self._initialize_system_message()

    def _setup_command_history(self):
        """Setup command history with persistent storage."""
        try:
            history_dir = pathlib.Path.home() / ".infragpt"
            history_dir.mkdir(exist_ok=True)
            history_file = history_dir / "history"

            # WARNING: FileHistory writes clear-text inputs to disk - security risk if users enter secrets
            self.prompt_session = PromptSession(history=FileHistory(str(history_file)))
            try:
                history_file.chmod(0o600)
            except Exception:
                pass

            if self.verbose:
                console.print(f"[dim]Command history: {history_file}[/dim]")

        except Exception as e:
            self.prompt_session = PromptSession()
            if self.verbose:
                console.print(
                    f"[dim]Warning: Could not setup command history: {e}[/dim]"
                )

    def _initialize_system_message(self):
        """Initialize the system message for the agent."""
        system_prompt = get_system_prompt()
        self.context.add_message("system", system_prompt)

    def run_interactive_session(self):
        """Run the main interactive agent session."""
        console.print(
            Panel.fit(
                "InfraGPT Shell Agent V2 - Direct SDK Integration",
                border_style="blue",
                title="[bold green]Shell Agent V2[/bold green]",
            )
        )

        console.print(f"[yellow]Model:[/yellow] [bold]{self.model_string}[/bold]")
        console.print("[dim]Validating API key...[/dim]")
        try:
            if self.llm_adapter.validate_api_key():
                console.print("[green]✓ API key validated[/green]")
            else:
                console.print("[red]✗ API key validation failed[/red]")
                return
        except Exception as e:
            console.print(f"[red]✗ API key validation failed: {e}[/red]")
            return

        console.print("[bold cyan]What would you like me to help with?[/bold cyan]")
        console.print(
            "[dim]Press Ctrl+D to exit, Ctrl+C to interrupt operations[/dim]\n"
        )

        while True:
            try:
                user_input = self._get_user_input()
                if not user_input:
                    continue

                if user_input.lower() in ["exit", "quit", "bye"]:
                    break

                self.context.add_message("user", user_input)
                self._process_user_input(user_input)

            except KeyboardInterrupt:
                continue
            except EOFError:
                console.print("\n[dim]EOF received (Ctrl+D). Exiting...[/dim]")
                break

        console.print("\n[bold]Goodbye![/bold]")

    def _get_user_input(self) -> str:
        """Get user input with prompt - use prompt_toolkit for proper interactive features."""
        try:
            return self.prompt_session.prompt("> ")
        except KeyboardInterrupt:
            return ""
        except EOFError:
            raise

    def _process_user_input(self, user_input: str):
        """Process user input with direct SDK streaming and tool execution."""
        try:
            messages = self.context.get_context_messages()

            if self.verbose:
                console.print(f"[dim]Context has {len(messages)} messages[/dim]")
                for i, msg in enumerate(messages):
                    role = msg.get("role", "unknown")
                    has_tools = "tool_calls" in msg
                    has_tool_id = "tool_call_id" in msg
                    content_len = len(str(msg.get("content", "")))
                    console.print(
                        f"[dim]  {i}: {role} (content: {content_len} chars, tools: {has_tools}, tool_id: {has_tool_id})[/dim]"
                    )

            console.print("\n[dim]Thinking...[/dim]")

            response_content = ""
            first_content = True

            try:
                for chunk in self.llm_adapter.stream_with_tools(messages):
                    if chunk.content:
                        if first_content:
                            console.print(
                                "\033[1A\033[K", end=""
                            )  # Move up and clear line
                            console.print("[bold green]A:[/bold green] ", end="")
                            first_content = False

                        response_content += chunk.content
                        console.print(chunk.content, end="")

                    if chunk.finish_reason:
                        if self.verbose:
                            console.print(
                                f"\n[dim]Finish reason: {chunk.finish_reason}[/dim]"
                            )
            except KeyboardInterrupt:
                console.print("\n[yellow]Operation cancelled by user.[/yellow]")
                return

            if response_content:
                console.print()
                self.context.add_message("assistant", response_content)

            self._log_interaction(user_input, response_content)

        except ToolExecutionCancelled:
            return
        except KeyboardInterrupt:
            return
        except Exception as e:
            console.print(f"[bold red]Error processing input:[/bold red] {e}")
            if self.verbose:
                import traceback

                console.print(traceback.format_exc())

    def _log_interaction(self, user_input: str, response: str):
        """Log the interaction for history. Sensitive fields are excluded explicitly."""
        try:
            interaction_data = {
                "user_input": user_input,
                "assistant_response": response,
                "model": self.model_string,
                "timestamp": datetime.utcnow().isoformat() + "Z",
                "verbose": self.verbose,
            }
            log_interaction("agent_conversation_v2", interaction_data)
        except Exception as e:
            if self.verbose:
                console.print(f"[dim]Warning: Could not log interaction: {e}[/dim]")


def run_shell_agent(model_string: str, api_key: str, verbose: bool = False):
    """Run the modern shell agent."""
    agent = ModernShellAgent(model_string, api_key, verbose)
    agent.run_interactive_session()
