"""Base agent classes for the Agent System."""

from abc import ABC, abstractmethod
from enum import Enum
from typing import Optional, TYPE_CHECKING
import logging

from src.models.agent import AgentResponse
from src.models.context import AgentContext

if TYPE_CHECKING:
    from src.llm import LiteLLMClient


class AgentType(Enum):
    """Agent types in the current system."""

    MAIN = "main"
    CONVERSATION = "conversation"
    RCA = "rca"


class BaseAgent(ABC):
    """Abstract base class for all agents."""

    def __init__(self, agent_type: AgentType):
        self.agent_type = agent_type
        self.logger = logging.getLogger(f"{__name__}.{agent_type.value}")

        # LLM client will be injected during initialization
        self.llm_client: Optional["LiteLLMClient"] = None

        self.logger.info(f"Initialized {agent_type.value} agent")

    @property
    def name(self) -> str:
        """Get the agent name."""
        return self.agent_type.value

    @abstractmethod
    async def can_handle(self, context: AgentContext) -> bool:
        """
        Determine if this agent can handle the given context.

        Args:
            context: The agent context to evaluate

        Returns:
            True if this agent can handle the request
        """
        pass

    @abstractmethod
    async def process(self, context: AgentContext) -> AgentResponse:
        """
        Process the request and generate a response.

        Args:
            context: The agent context to process

        Returns:
            The agent response
        """
        pass

    def set_llm_client(self, llm_client: "LiteLLMClient") -> None:
        """Set the LLM client for this agent."""
        self.llm_client = llm_client
        self.logger.debug(f"Set LLM client for {self.name} agent")

    def __str__(self) -> str:
        return f"{self.agent_type.value.title()}Agent"
