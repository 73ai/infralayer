"""
Unified exception hierarchy for LLM providers.
"""

from typing import Optional


class LLMError(Exception):
    """Base exception for all LLM-related errors."""

    def __init__(
        self,
        message: str,
        provider: Optional[str] = None,
        model: Optional[str] = None,
    ):
        self.provider = provider
        self.model = model
        super().__init__(message)


class AuthenticationError(LLMError):
    """API key authentication failed."""

    pass


class RateLimitError(LLMError):
    """Rate limit exceeded."""

    pass


class APIError(LLMError):
    """General API error."""

    def __init__(
        self,
        message: str,
        status_code: Optional[int] = None,
        provider: Optional[str] = None,
        model: Optional[str] = None,
    ):
        self.status_code = status_code
        super().__init__(message, provider, model)


class ToolCallError(LLMError):
    """Tool call related error."""

    pass


class ValidationError(LLMError):
    """Input validation error."""

    pass


class ContextWindowError(LLMError):
    """Context window exceeded."""

    pass
