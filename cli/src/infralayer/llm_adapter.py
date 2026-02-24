"""
LLM adapter for unified provider access.
"""

import json
from typing import Iterator, List, Dict, Any
from rich.console import Console

from .llm import LLMRouter, StreamChunk, ToolCall
from .llm.exceptions import ValidationError, LLMError
from .tools import get_available_tools, execute_tool_call, ToolExecutionCancelled


console = Console()


class LLMAdapter:
    """LLM adapter for streaming responses and tool calling."""

    def __init__(self, model_string: str, api_key: str, verbose: bool = False):
        """
        Initialize LLM adapter.

        Args:
            model_string: Provider:model format (e.g., "openai:gpt-4o")
            api_key: API key for the provider
            verbose: Enable verbose logging
        """
        self.model_string = model_string
        self.api_key = api_key
        self.verbose = verbose

        try:
            self.provider = LLMRouter.create_provider(model_string, api_key)
        except Exception as e:
            raise ValidationError(f"Failed to initialize LLM provider: {e}") from e

    def validate_api_key(self) -> bool:
        """Validate API key."""
        try:
            return self.provider.validate_api_key()
        except Exception as e:
            if self.verbose:
                console.print(f"[red]API key validation failed: {e}[/red]")
            return False

    def stream_with_tools(
        self, messages: List[Dict[str, Any]]
    ) -> Iterator[StreamChunk]:
        """
        Stream chat with tool support.

        Args:
            messages: List of messages in OpenAI format

        Yields:
            StreamChunk objects with content and/or tool calls
        """
        try:
            tools = get_available_tools()
            tool_calls_buffer = []

            try:
                for chunk in self.provider.stream(messages, tools=tools):
                    if chunk.content:
                        yield chunk

                    if chunk.tool_calls:
                        if self.verbose:
                            console.print(
                                f"[dim]Tool calls: {[tc.name for tc in chunk.tool_calls]}[/dim]"
                            )
                        tool_calls_buffer.extend(chunk.tool_calls)
                        yield chunk

                    if chunk.finish_reason:
                        if self.verbose:
                            console.print(
                                f"[dim]Finish reason: {chunk.finish_reason}[/dim]"
                            )
                        yield chunk
            except KeyboardInterrupt:
                console.print("\n[yellow]Streaming cancelled by user.[/yellow]")
                return

            if tool_calls_buffer:
                try:
                    yield from self._execute_tool_calls(tool_calls_buffer, messages)
                except KeyboardInterrupt:
                    console.print(
                        "\n[yellow]Tool execution cancelled by user.[/yellow]"
                    )
                    return

        except ToolExecutionCancelled:
            raise
        except Exception as e:
            error_msg = f"Streaming failed: {e}"
            if self.verbose:
                console.print(f"[red]{error_msg}[/red]")
            raise LLMError(error_msg) from e

    def _execute_tool_calls(
        self, tool_calls: List[ToolCall], original_messages: List[Dict[str, Any]]
    ) -> Iterator[StreamChunk]:
        """Execute tool calls and continue conversation."""
        provider_name, _ = LLMRouter.parse_model_string(self.model_string)
        tool_results = []
        for tool_call in tool_calls:
            try:
                if self.verbose:
                    console.print(
                        f"[dim]Executing tool: {tool_call.name} with args: {tool_call.arguments}[/dim]"
                    )

                result = execute_tool_call(tool_call.name, tool_call.arguments)
                tool_results.append(
                    {
                        "tool_call_id": tool_call.id,
                        "name": tool_call.name,
                        "content": result,
                    }
                )

            except ToolExecutionCancelled:
                raise
            except Exception as e:
                error_msg = f"Tool execution failed: {e}"
                console.print(f"[red]{error_msg}[/red]")
                if self.verbose:
                    import traceback

                    console.print(f"[dim]Traceback: {traceback.format_exc()}[/dim]")
                tool_results.append(
                    {
                        "tool_call_id": tool_call.id,
                        "name": tool_call.name,
                        "content": error_msg,
                    }
                )

        updated_messages = original_messages.copy()

        if provider_name == "anthropic":
            assistant_content = []
            for tc in tool_calls:
                assistant_content.append(
                    {
                        "type": "tool_use",
                        "id": tc.id,
                        "name": tc.name,
                        "input": tc.arguments,
                    }
                )

            updated_messages.append({"role": "assistant", "content": assistant_content})

            user_content = []
            for result in tool_results:
                user_content.append(
                    {
                        "type": "tool_result",
                        "tool_use_id": result["tool_call_id"],
                        "content": result["content"],
                    }
                )

            updated_messages.append({"role": "user", "content": user_content})

        else:
            tool_call_message = {
                "role": "assistant",
                "content": None,
                "tool_calls": [
                    {
                        "id": tc.id,
                        "type": "function",
                        "function": {
                            "name": tc.name,
                            "arguments": tc.arguments
                            if isinstance(tc.arguments, str)
                            else json.dumps(tc.arguments),
                        },
                    }
                    for tc in tool_calls
                ],
            }
            updated_messages.append(tool_call_message)

            for result in tool_results:
                updated_messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": result["tool_call_id"],
                        "name": result["name"],
                        "content": result["content"],
                    }
                )

        if self.verbose:
            console.print("[dim]Continuing conversation after tool execution...[/dim]")

        try:
            yield from self.stream_with_tools(updated_messages)

        except Exception as e:
            error_msg = f"Follow-up conversation failed: {e}"
            console.print(f"[red]{error_msg}[/red]")
            yield StreamChunk(content=f"Error: {error_msg}", finish_reason="error")


def get_llm_adapter(
    model_string: str, api_key: str, verbose: bool = False
) -> LLMAdapter:
    """
    Create LLM adapter instance.

    Args:
        model_string: Provider:model format (e.g., "openai:gpt-4o")
        api_key: API key for the provider
        verbose: Enable verbose logging

    Returns:
        Configured LLM adapter
    """
    return LLMAdapter(model_string, api_key, verbose)
