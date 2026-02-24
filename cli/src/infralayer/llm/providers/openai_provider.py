"""
OpenAI provider implementation using direct SDK.
"""

import json
from typing import List, Dict, Any, Iterator, Optional
from openai import OpenAI
import openai

from ..base import BaseLLMProvider
from ..models import StreamChunk, ToolCall, Tool
from ..exceptions import (
    AuthenticationError,
    RateLimitError,
    APIError,
    ContextWindowError,
    ValidationError,
)


class OpenAIProvider(BaseLLMProvider):
    """OpenAI provider using official Python SDK."""

    def _initialize_client(self, **kwargs: Any) -> OpenAI:
        """Initialize OpenAI client."""
        return OpenAI(api_key=self.api_key)

    def validate_api_key(self) -> bool:
        """Validate API key with a simple test call."""
        try:
            params = {
                "model": self.model,
                "messages": [{"role": "user", "content": "hi"}],
            }

            # Newer models like o4-mini use max_completion_tokens
            if (
                self.model.startswith("o4")
                or self.model.startswith("o1")
                or self.model.startswith("gpt-5")
            ):
                params["max_completion_tokens"] = 10
            else:
                params["max_tokens"] = 10

            self._client.chat.completions.create(**params)
            return True
        except Exception as e:
            raise self._map_error(e)

    def stream(
        self,
        messages: List[Dict[str, Any]],
        tools: Optional[List[Tool]] = None,
        **kwargs: Any,
    ) -> Iterator[StreamChunk]:
        """Stream response with unified tool calling support."""
        try:
            request_params = self._build_request(messages, tools, **kwargs)
            response = self._client.chat.completions.create(**request_params)

            # Buffer for tool calls - persistent across chunks
            tool_call_buffer = {}
            accumulated_tool_calls = []
            call_id_by_index = {}  # Track call IDs per index for continuations

            for chunk in response:
                if chunk.choices and len(chunk.choices) > 0:
                    choice = chunk.choices[0]

                    content = None
                    if choice.delta.content:
                        content = choice.delta.content

                    tool_calls = None
                    if choice.delta.tool_calls:
                        for tc in choice.delta.tool_calls:
                            if tc.id and hasattr(tc, "index") and tc.index is not None:
                                call_id_by_index[tc.index] = tc.id

                        tool_calls = self._process_tool_calls(
                            choice.delta.tool_calls, tool_call_buffer, call_id_by_index
                        )
                        if tool_calls:
                            accumulated_tool_calls.extend(tool_calls)

                    finish_reason = choice.finish_reason

                    if content or (finish_reason and finish_reason != "tool_calls"):
                        yield StreamChunk(
                            content=content,
                            tool_calls=None,  # Don't yield tool calls during streaming
                            finish_reason=finish_reason
                            if finish_reason != "tool_calls"
                            else None,
                        )

                    # Only emit tool calls once when we get the finish reason
                    if finish_reason == "tool_calls" and accumulated_tool_calls:
                        yield StreamChunk(
                            content=None,
                            tool_calls=accumulated_tool_calls,
                            finish_reason=finish_reason,
                        )
                        accumulated_tool_calls = []

        except Exception as e:
            raise self._map_error(e)

    def _build_request(
        self, messages: List[Dict[str, Any]], tools: Optional[List[Tool]], **kwargs: Any
    ) -> Dict[str, Any]:
        """Build OpenAI API request."""
        request = {
            "model": self.model,
            "messages": messages,
            "stream": True,
        }

        # o4/o1/gpt-5 models only support default temperature of 1.0
        if (
            self.model.startswith("o4")
            or self.model.startswith("o1")
            or self.model.startswith("gpt-5")
        ):
            pass
        else:
            request["temperature"] = kwargs.get("temperature", 0.0)

        if kwargs.get("max_tokens"):
            if (
                self.model.startswith("o4")
                or self.model.startswith("o1")
                or self.model.startswith("gpt-5")
            ):
                request["max_completion_tokens"] = kwargs["max_tokens"]
            else:
                request["max_tokens"] = kwargs["max_tokens"]

        if tools:
            request["tools"] = self._convert_tools(tools)
            request["tool_choice"] = "auto"

        return request

    def _convert_messages(self, messages: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Convert unified message format to OpenAI format."""
        # OpenAI format is already our unified format
        return messages

    def _convert_tools(self, tools: List[Tool]) -> List[Dict[str, Any]]:
        """Convert Tool objects to OpenAI format."""

        openai_tools = []
        for tool in tools:
            parameters_dict = {
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

            openai_tools.append(
                {
                    "type": "function",
                    "function": {
                        "name": tool.name,
                        "description": tool.description,
                        "parameters": parameters_dict,
                    },
                }
            )

        return openai_tools

    def _process_tool_calls(
        self, delta_tool_calls, buffer, call_id_by_index
    ) -> Optional[List[ToolCall]]:
        """Process streaming tool calls."""
        completed_tools = []

        for i, delta_call in enumerate(delta_tool_calls):
            call_id = delta_call.id
            if not call_id:
                index = getattr(delta_call, "index", None)
                if index is not None and index in call_id_by_index:
                    call_id = call_id_by_index[index]
                else:
                    continue
            elif call_id not in buffer:
                buffer[call_id] = {
                    "id": call_id,
                    "name": "",
                    "arguments": "",
                    "complete": False,
                }

            if delta_call.function:
                if delta_call.function.name:
                    buffer[call_id]["name"] = delta_call.function.name
                if delta_call.function.arguments:
                    buffer[call_id]["arguments"] += delta_call.function.arguments

            tool_data = buffer[call_id]
            if (
                not buffer[call_id]["complete"]
                and tool_data["name"]
                and tool_data["arguments"]
            ):
                try:
                    arguments = json.loads(tool_data["arguments"])
                    buffer[call_id]["complete"] = True
                    completed_tools.append(
                        ToolCall(
                            id=tool_data["id"],
                            name=tool_data["name"],
                            arguments=arguments,
                        )
                    )
                except json.JSONDecodeError:
                    pass

        return completed_tools if completed_tools else None

    def _map_error(self, error: Exception) -> Exception:
        """Map OpenAI errors to unified exceptions."""
        if isinstance(error, openai.AuthenticationError):
            return AuthenticationError(str(error), provider="openai", model=self.model)
        elif isinstance(error, openai.RateLimitError):
            return RateLimitError(str(error), provider="openai", model=self.model)
        elif isinstance(error, openai.BadRequestError):
            if "context window" in str(error).lower():
                return ContextWindowError(
                    str(error), provider="openai", model=self.model
                )
            return ValidationError(str(error), provider="openai", model=self.model)
        elif isinstance(error, openai.APIStatusError):
            return APIError(
                str(error),
                status_code=error.status_code,
                provider="openai",
                model=self.model,
            )
        else:
            return APIError(
                f"OpenAI API error: {error}", provider="openai", model=self.model
            )
