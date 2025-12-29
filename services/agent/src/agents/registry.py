"""Agent registry and factory for managing agent instances."""

import logging
from typing import Any, Dict, Optional

from src.agents.main_agent import MainAgent
from src.models.agent import AgentRequest, AgentResponse
from src.models.context import AgentContext
from src.llm import LiteLLMClient


class AgentSystem:
    """Central system for managing and coordinating all agents."""

    def __init__(self, llm_client: LiteLLMClient = None):
        self.logger = logging.getLogger(__name__)
        self.llm_client = llm_client or LiteLLMClient()
        self.main_agent: Optional[MainAgent] = None
        self._initialized = False

    async def initialize(self) -> None:
        """Initialize the agent system."""
        if self._initialized:
            self.logger.warning("Agent system already initialized")
            return

        try:
            self.logger.info("Initializing agent system...")

            # Create main agent with shared LLM client (which creates sub-agents)
            self.main_agent = MainAgent(self.llm_client)

            self._initialized = True
            self.logger.info("Agent system initialized successfully")

        except Exception as e:
            self.logger.error(f"Failed to initialize agent system: {e}")
            raise

    async def process_request(self, request: AgentRequest) -> AgentResponse:
        """
        Process an agent request through the agent system.

        Args:
            request: The agent request to process

        Returns:
            The agent response
        """
        if not self._initialized or not self.main_agent:
            raise RuntimeError("Agent system not initialized")

        try:
            # Convert request to context
            context = self._create_context(request)

            # Process through main agent
            response = await self.main_agent.process(context)

            self.logger.info(
                f"Request processed successfully by {response.agent_type} agent "
                f"with confidence {response.confidence}"
            )

            return response

        except Exception as e:
            self.logger.error(f"Error processing agent request: {e}")
            return AgentResponse(
                success=False,
                response_text="I encountered an error while processing your request.",
                error_message=str(e),
                agent_type="system",
                confidence=0.0,
                tools_used=[],
            )

    def _create_context(self, request: AgentRequest) -> AgentContext:
        """Create agent context from request."""
        return AgentContext(
            conversation_id=request.conversation_id,
            user_id=request.user_id,
            channel_id=request.channel_id,
            current_message=request.current_message,
            message_history=request.past_messages,
            metadata={"context": request.context},
        )

    def set_llm_client(self, llm_client: LiteLLMClient) -> None:
        """Set LLM client for the agent system."""
        if self.main_agent:
            self.main_agent.set_llm_client(llm_client)
            self.logger.info("LLM client set for agent system")
        else:
            self.logger.warning("Cannot set LLM client - agent system not initialized")

    def is_ready(self) -> bool:
        """Check if the agent system is ready to process requests."""
        return self._initialized and self.main_agent is not None

    def get_system_status(self) -> Dict[str, Any]:
        """Get status of the entire agent system."""
        if not self._initialized:
            return {"status": "not_initialized"}

        status = {
            "status": "ready" if self.is_ready() else "not_ready",
            "initialized": self._initialized,
            "main_agent_available": self.main_agent is not None,
        }

        if self.main_agent:
            status["agents"] = self.main_agent.get_agent_status()

        return status
