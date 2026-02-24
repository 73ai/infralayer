"""
Router layer for provider selection and model configuration.
"""

from typing import Dict, Any, Tuple
from .providers import OpenAIProvider, AnthropicProvider
from .base import BaseLLMProvider
from .exceptions import ValidationError


class LLMRouter:
    """Router for selecting and configuring LLM providers."""

    MODEL_CONFIGS = {
        "openai": {
            "default_params": {"temperature": 0.0},
            "provider_class": OpenAIProvider,
        },
        "anthropic": {
            "default_params": {"temperature": 0.0, "max_tokens": 4096},
            "provider_class": AnthropicProvider,
        },
    }

    @classmethod
    def parse_model_string(cls, model_string: str) -> Tuple[str, str]:
        """Parse provider:model string format."""
        if ":" not in model_string:
            raise ValidationError(
                f"Invalid model format '{model_string}'. Expected format: 'provider:model' "
                f"(e.g., 'openai:gpt-4o', 'anthropic:claude-3-5-sonnet-20241022')"
            )

        parts = model_string.split(":", 1)
        provider = parts[0].lower()
        model = parts[1]

        if provider not in cls.MODEL_CONFIGS:
            available_providers = list(cls.MODEL_CONFIGS.keys())
            raise ValidationError(
                f"Unknown provider '{provider}'. Available providers: {available_providers}"
            )

        return provider, model

    @classmethod
    def create_provider(
        cls, model_string: str, api_key: str, **kwargs
    ) -> BaseLLMProvider:
        """Create provider instance from model string."""
        provider_name, model = cls.parse_model_string(model_string)

        config = cls.MODEL_CONFIGS[provider_name]
        provider_class = config["provider_class"]
        merged_kwargs = {**config["default_params"], **kwargs}

        try:
            return provider_class(api_key=api_key, model=model, **merged_kwargs)
        except Exception as e:
            raise ValidationError(f"Failed to create {provider_name} provider: {e}")

    @classmethod
    def validate_model_string(cls, model_string: str) -> bool:
        """Validate model string format."""
        try:
            cls.parse_model_string(model_string)
            return True
        except ValidationError:
            return False

    @classmethod
    def get_supported_providers(cls) -> Dict[str, Dict[str, Any]]:
        """Get information about supported providers."""
        return {
            provider: {
                "default_params": config["default_params"],
                "class_name": config["provider_class"].__name__,
            }
            for provider, config in cls.MODEL_CONFIGS.items()
        }

    @classmethod
    def get_provider_examples(cls) -> Dict[str, str]:
        """Get example model strings for each provider."""
        return {
            "openai": "openai:gpt-4o",
            "anthropic": "anthropic:claude-3-5-sonnet-20241022",
        }
