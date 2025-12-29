#!/usr/bin/env python3

import os
import sys
from typing import Optional, Tuple

import click
from rich.panel import Panel

from infragpt.config import init_config, console
from infragpt.llm.router import LLMRouter
from infragpt.llm.exceptions import ValidationError, AuthenticationError
from infragpt.history import history_command
from infragpt.agent import run_shell_agent
from infragpt.container import (
    is_sandbox_mode,
    get_executor,
    cleanup_executor,
    cleanup_old_containers,
    DockerNotAvailableError,
)
from infragpt.tools import cleanup_executor as cleanup_tools_executor
from infragpt.auth import (
    login as auth_login,
    logout as auth_logout,
    get_auth_status,
    is_authenticated,
    validate_token_with_api,
    refresh_token_strict,
    fetch_gcp_credentials_strict,
    fetch_gke_cluster_info,
    write_gcp_credentials_file,
    cleanup_credentials,
)
from infragpt.exceptions import (
    AuthValidationError,
    TokenRefreshError,
    GCPCredentialError,
    ContainerSetupError,
)


@click.group(invoke_without_command=True)
@click.pass_context
@click.version_option(package_name="infragpt")
@click.option(
    "--model",
    "-m",
    help="Model in provider:model format (e.g., openai:gpt-4o, anthropic:claude-3-5-sonnet-20241022)",
)
@click.option("--api-key", "-k", help="API key for the selected provider")
@click.option("--verbose", "-v", is_flag=True, help="Enable verbose output")
def cli(ctx, model, api_key, verbose):
    """InfraGPT V2 - Interactive shell operations with direct SDK integration."""
    if ctx.invoked_subcommand is None:
        main(model=model, api_key=api_key, verbose=verbose)


@cli.command(name="history")
@click.option(
    "--limit", "-l", type=int, default=10, help="Number of history entries to display"
)
@click.option("--type", "-t", help="Filter by interaction type")
@click.option("--export", "-e", help="Export history to file path")
def history_cli(limit, type, export):
    """View or export interaction history."""
    history_command(limit, type, export)


@cli.command(name="providers")
def providers_cli():
    """Show supported providers and example model strings."""
    console.print(
        Panel.fit(
            "Supported Providers and Model Examples",
            border_style="blue",
            title="[bold green]Providers[/bold green]",
        )
    )

    providers = LLMRouter.get_supported_providers()
    examples = LLMRouter.get_provider_examples()

    for provider, config in providers.items():
        console.print(f"\n[bold cyan]{provider.upper()}[/bold cyan]")
        console.print(f"  Example: [yellow]{examples[provider]}[/yellow]")
        console.print(f"  Default params: {config['default_params']}")


@cli.group()
def auth():
    """Authentication commands for InfraGPT platform."""
    pass


@auth.command(name="login")
def auth_login_cli():
    """Authenticate with InfraGPT platform."""
    auth_login()


@auth.command(name="logout")
def auth_logout_cli():
    """Remove stored credentials and revoke token."""
    auth_logout()


@auth.command(name="status")
def auth_status_cli():
    """Show authentication status."""
    status = get_auth_status()

    if not status.authenticated:
        console.print("[yellow]Not authenticated.[/yellow]")
        console.print("\nRun [cyan]infragpt auth login[/cyan] to authenticate.")
        return

    console.print("[green]Authenticated[/green]\n")

    if status.organization_id:
        console.print(f"Organization ID: [cyan]{status.organization_id}[/cyan]")
    if status.user_id:
        console.print(f"User ID: [cyan]{status.user_id}[/cyan]")
    if status.expires_at:
        console.print(f"Token expires: [dim]{status.expires_at}[/dim]")


def get_credentials_v2(
    model_string: Optional[str] = None,
    api_key: Optional[str] = None,
    verbose: bool = False,
) -> Tuple[str, str]:
    """Get credentials for the new system."""
    if model_string:
        if not LLMRouter.validate_model_string(model_string):
            raise ValidationError("Invalid model format. Use 'provider:model' format.")

        provider_name, model_name = LLMRouter.parse_model_string(model_string)
    else:
        provider_name = None

    if not api_key:
        if provider_name == "openai":
            api_key = os.getenv("OPENAI_API_KEY")
        elif provider_name == "anthropic":
            api_key = os.getenv("ANTHROPIC_API_KEY")
        else:
            openai_key = os.getenv("OPENAI_API_KEY")
            anthropic_key = os.getenv("ANTHROPIC_API_KEY")

            if openai_key and not model_string:
                model_string = "openai:gpt-5"
                api_key = openai_key
                provider_name = "openai"
            elif anthropic_key and not model_string:
                model_string = "anthropic:claude-sonnet-4-20250514"
                api_key = anthropic_key
                provider_name = "anthropic"

    if not model_string or not api_key:
        console.print(
            "\n[yellow]No valid credentials found. Please provide model and API key.[/yellow]"
        )

        if not model_string:
            console.print("\nSupported formats:")
            examples = LLMRouter.get_provider_examples()
            for provider, example in examples.items():
                console.print(f"  {provider}: [cyan]{example}[/cyan]")

            while True:
                model_input = console.input("\nEnter model (provider:model): ").strip()
                if LLMRouter.validate_model_string(model_input):
                    model_string = model_input
                    provider_name, _ = LLMRouter.parse_model_string(model_string)
                    break
                else:
                    console.print(
                        "[red]Invalid format. Please use 'provider:model' format.[/red]"
                    )

        if not api_key:
            api_key = console.input(f"Enter API key for {provider_name}: ").strip()

    return model_string, api_key


def main(model: Optional[str], api_key: Optional[str], verbose: bool) -> None:
    """InfraGPT V2 - Interactive shell operations with direct SDK integration."""
    init_config()

    if verbose:
        from importlib.metadata import version

        try:
            console.print(f"[dim]InfraGPT V2 version: {version('infragpt')}[/dim]")
        except Exception:
            console.print("[dim]InfraGPT V2: Version information not available[/dim]")

    sandbox_started = False
    gcp_creds_path = None

    try:
        authenticated = is_authenticated()
        sandbox = is_sandbox_mode()
        gke_cluster = None

        if sandbox and not authenticated:
            console.print("[yellow]Sandbox mode requires authentication.[/yellow]")
            console.print("\nRun [cyan]infragpt auth login[/cyan] to authenticate.")
            sys.exit(1)

        if authenticated and sandbox:
            # STRICT MODE - all failures exit CLI
            validate_token_with_api()
            refresh_token_strict()
            gcp_creds = fetch_gcp_credentials_strict()
            gke_cluster = fetch_gke_cluster_info()
            gcp_creds_path = write_gcp_credentials_file(gcp_creds)
            if verbose:
                console.print("[dim]GCP credentials loaded.[/dim]")
                if gke_cluster:
                    console.print(f"[dim]GKE cluster: {gke_cluster.cluster_name}[/dim]")
                else:
                    console.print("[dim]GKE cluster: auto-discover[/dim]")

        if sandbox:
            removed = cleanup_old_containers()
            if removed > 0:
                console.print(
                    f"[dim]Cleaned up {removed} old sandbox container(s)[/dim]"
                )
            console.print(
                "[yellow]Sandbox mode enabled - starting Docker container...[/yellow]"
            )
            executor = get_executor(
                gcp_credentials_path=gcp_creds_path,
                gke_cluster_info=gke_cluster,
            )
            executor.start()
            sandbox_started = True
            if gcp_creds_path:
                console.print(
                    "[green]Sandbox container ready (GCP configured).[/green]\n"
                )
            else:
                console.print("[green]Sandbox container ready.[/green]\n")

        model_string, resolved_api_key = get_credentials_v2(model, api_key, verbose)

        if verbose:
            console.print(f"[dim]Using model: {model_string}[/dim]")

        run_shell_agent(model_string, resolved_api_key, verbose)

    except (AuthValidationError, TokenRefreshError) as e:
        console.print(f"[red]Authentication Error: {e}[/red]")
        console.print("\nRun [cyan]infragpt auth login[/cyan] to re-authenticate.")
        sys.exit(1)
    except GCPCredentialError as e:
        console.print(f"[red]Credential Error: {e}[/red]")
        sys.exit(1)
    except (DockerNotAvailableError, ContainerSetupError) as e:
        console.print(f"[red]Error: {e}[/red]")
        console.print(
            "Please fix the issue above or disable sandbox mode with INFRAGPT_ISOLATED=false"
        )
        sys.exit(1)
    except ValidationError as e:
        console.print(f"[red]Validation Error: {e}[/red]")
        console.print(
            "\nUse --help to see usage information or run 'infragpt providers' to see supported providers."
        )
        sys.exit(1)
    except AuthenticationError as e:
        console.print(f"[red]Authentication Error: {e}[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        if verbose:
            import traceback

            console.print(traceback.format_exc())
        sys.exit(1)
    finally:
        if sandbox_started:
            cleanup_executor()
            cleanup_tools_executor()
        if gcp_creds_path:
            cleanup_credentials()


if __name__ == "__main__":
    cli()
