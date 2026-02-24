"""
Anthropic provider implementation using direct SDK.
"""

import json
import logging
from typing import List, Dict, Any, Iterator, Optional
from anthropic import Anthropic
import anthropic

from ..base import BaseLLMProvider
from ..models import StreamChunk, ToolCall, Tool
from ..exceptions import (
    AuthenticationError,
    RateLimitError,
    APIError,
    ContextWindowError,
    ValidationError,
)

logger = logging.getLogger(__name__)


class AnthropicProvider(BaseLLMProvider):
    """Anthropic provider using official Python SDK.

    Attributes:
        api_key: The Anthropic API key for authentication.
        model: The model identifier to use (e.g., 'claude-3-opus-20240229').
        provider_name: The provider name (set to 'anthropic').
    """

    def _initialize_client(self, **kwargs: Any) -> Anthropic:
        """Initialize Anthropic client."""
        return Anthropic(api_key=self.api_key)

    def validate_api_key(self) -> bool:
        """Validate API key with a simple test call."""
        try:
            self._client.messages.create(
                model=self.model,
                max_tokens=1,
                messages=[{"role": "user", "content": "hi"}],
            )
            return True
        except Exception as e:
            raise self._map_error(e) from e

    def stream(
        self,
        messages: List[Dict[str, Any]],
        tools: Optional[List[Tool]] = None,
        **kwargs: Any,
    ) -> Iterator[StreamChunk]:
        """Stream response with unified tool calling support following Anthropic best practices."""
        try:
            request_params = self._build_request(messages, tools, **kwargs)
            response = self._client.messages.create(**request_params)

            # Buffer for tool calls using index-based tracking (Anthropic best practice)
            tool_use_inputs = {}  # index -> accumulated JSON string
            tool_blocks = {}  # index -> tool metadata

            for event in response:
                chunk = None

                if event.type == "content_block_start":
                    if event.content_block.type == "tool_use":
                        index = event.index
                        tool_blocks[index] = {
                            "id": event.content_block.id,
                            "name": event.content_block.name,
                        }
                        tool_use_inputs[index] = ""

                elif event.type == "content_block_delta":
                    if event.delta.type == "text_delta":
                        chunk = StreamChunk(content=event.delta.text)
                    elif event.delta.type == "input_json_delta":
                        index = event.index
                        if index in tool_use_inputs:
                            tool_use_inputs[index] += event.delta.partial_json

                elif event.type == "content_block_stop":
                    index = event.index
                    if index in tool_blocks and index in tool_use_inputs:
                        tool_block = tool_blocks[index]
                        json_str = tool_use_inputs[index]

                        try:
                            arguments = json.loads(json_str) if json_str else {}
                            tool_call = ToolCall(
                                id=tool_block["id"],
                                name=tool_block["name"],
                                arguments=arguments,
                            )
                            yield StreamChunk(tool_calls=[tool_call])

                        except json.JSONDecodeError as e:
                            logger.warning(
                                f"Failed to parse JSON for {tool_block['name']}: {json_str} - Error: {e}"
                            )

                elif event.type == "message_delta" and hasattr(
                    event.delta, "stop_reason"
                ):
                    chunk = StreamChunk(finish_reason=event.delta.stop_reason)

                if chunk:
                    yield chunk

        except Exception as e:
            raise self._map_error(e) from e

    def _build_request(
        self,
        messages: List[Dict[str, Any]],
        tools: Optional[List[Tool]] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Build Anthropic API request."""
        system_message = None
        filtered_messages = []

        for message in messages:
            if message["role"] == "system":
                system_message = message["content"]
            else:
                filtered_messages.append(message)

        request = {
            "model": self.model,
            "messages": filtered_messages,
            "max_tokens": kwargs.get("max_tokens", 4096),  # Required for Anthropic
            "stream": True,
            "temperature": kwargs.get("temperature", 0.0),
        }

        if system_message:
            request["system"] = system_message

        if tools:
            request["tools"] = self._convert_tools(tools)

        return request

    def _convert_messages(self, messages: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Convert unified message format to Anthropic format."""
        converted_messages = []
        system_message = None

        for message in messages:
            if message["role"] == "system":
                system_message = message["content"]
            else:
                converted_messages.append(message)

        return {"messages": converted_messages, "system": system_message}

    def _convert_tools(self, tools: List[Tool]) -> List[Dict[str, Any]]:
        """Convert Tool objects to Anthropic format."""
        anthropic_tools = []
        for tool in tools:
            input_schema_dict = {
                "type": tool.input_schema.type,
                "properties": {
                    name: {
                        "type": param.type,
                        "description": param.description,
                        **({"enum": param.enum} if param.enum else {}),
                        **(
                            {"default": param.default}
                            if param.default is not None
                            else {}
                        ),
                    }
                    for name, param in tool.input_schema.properties.items()
                },
                "required": tool.input_schema.required,
                "additionalProperties": tool.input_schema.additionalProperties,
            }

            anthropic_tools.append(
                {
                    "name": tool.name,
                    "description": tool.description,
                    "input_schema": input_schema_dict,
                }
            )

        return anthropic_tools

    def _map_error(self, error: Exception) -> Exception:
        """Map Anthropic errors to unified exceptions."""
        if isinstance(error, anthropic.AuthenticationError):
            return AuthenticationError(
                str(error), provider="anthropic", model=self.model
            )
        elif isinstance(error, anthropic.RateLimitError):
            return RateLimitError(str(error), provider="anthropic", model=self.model)
        elif isinstance(error, anthropic.BadRequestError):
            if (
                "context window" in str(error).lower()
                or "too long" in str(error).lower()
            ):
                return ContextWindowError(
                    str(error), provider="anthropic", model=self.model
                )
            return ValidationError(str(error), provider="anthropic", model=self.model)
        elif isinstance(error, anthropic.APIStatusError):
            return APIError(
                str(error),
                status_code=error.status_code,
                provider="anthropic",
                model=self.model,
            )
        else:
            return APIError(
                f"Anthropic API error: {error}", provider="anthropic", model=self.model
            )
