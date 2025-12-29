"""Main orchestrator agent that routes requests to appropriate sub-agents."""

from typing import Dict, List, Optional

from src.agents.base import BaseAgent, AgentType
from src.agents.conversation import ConversationAgent
from src.agents.rca import RCAAgent
from src.models.agent import AgentResponse
from src.models.context import AgentContext
from src.llm import LiteLLMClient


class MainAgent(BaseAgent):
    """Main orchestrator agent that routes requests to sub-agents."""

    def __init__(self, llm_client: LiteLLMClient = None):
        super().__init__(AgentType.MAIN)

        # Initialize shared LLM client
        self.llm_client = llm_client or LiteLLMClient()

        # Initialize sub-agents with shared LLM client
        self.conversation_agent = ConversationAgent(self.llm_client)
        self.rca_agent = RCAAgent(self.llm_client)

        # Create agent registry
        self.sub_agents: List[BaseAgent] = [
            self.rca_agent,  # RCA first - more specific
            self.conversation_agent,  # Conversation as fallback
        ]

        self.logger.info(
            f"Initialized main agent with {len(self.sub_agents)} sub-agents"
        )

    async def can_handle(self, context: AgentContext) -> bool:
        """Main agent can handle any request by routing to sub-agents."""
        return True

    async def process(self, context: AgentContext) -> AgentResponse:
        """Route request to appropriate sub-agent and process."""
        self.logger.info(
            f"Main agent processing request for conversation: {context.conversation_id}"
        )

        try:
            # Select the best sub-agent for this request
            selected_agent = await self._select_agent(context)

            if not selected_agent:
                return self._create_fallback_response(context)

            # Update context with selected agent
            context.selected_agent = selected_agent.name

            self.logger.info(f"Routing to {selected_agent.name} agent")

            # Process with selected agent
            response = await selected_agent.process(context)

            # Add routing information to response
            if response.success:
                self.logger.info(
                    f"Successfully processed by {selected_agent.name} agent"
                )
            else:
                self.logger.warning(
                    f"Processing failed in {selected_agent.name} agent: {response.error_message}"
                )

            return response

        except Exception as e:
            self.logger.error(f"Error in main agent processing: {e}")
            return AgentResponse(
                success=False,
                response_text="I encountered an internal error while processing your request.",
                error_message=str(e),
                agent_type=self.name,
                confidence=0.0,
                tools_used=[],
            )

    async def _select_agent(self, context: AgentContext) -> Optional[BaseAgent]:
        """
        Select the most appropriate sub-agent for the given context.

        Agents are evaluated in order of specificity:
        1. RCA Agent - for technical issues and troubleshooting
        2. Conversation Agent - for general interactions
        """
        self.logger.debug(
            f"Selecting agent for message: {context.current_message[:100]}..."
        )

        for agent in self.sub_agents:
            try:
                if await agent.can_handle(context):
                    self.logger.debug(f"Selected {agent.name} agent")
                    return agent
            except Exception as e:
                self.logger.error(f"Error checking {agent.name} agent capability: {e}")
                continue

        self.logger.warning("No suitable agent found for request")
        return None

    def _create_fallback_response(self, context: AgentContext) -> AgentResponse:
        """Create a fallback response when no agent can handle the request."""
        return AgentResponse(
            success=True,
            response_text=(
                "I'm not sure how to best help with that specific request. "
                "Could you try rephrasing it or providing more details? "
                "I'm designed to help with general questions and troubleshooting technical issues."
            ),
            agent_type=self.name,
            confidence=0.3,
            tools_used=[],
        )

    def set_llm_client(self, llm_client: LiteLLMClient) -> None:
        """Set LLM client for all agents."""
        super().set_llm_client(llm_client)

        # Propagate to sub-agents
        for agent in self.sub_agents:
            agent.set_llm_client(llm_client)

        self.logger.debug("Set LLM client for all sub-agents")

    def get_agent_status(self) -> Dict[str, str]:
        """Get status of all agents."""
        return {"main": "active", "conversation": "active", "rca": "active"}
