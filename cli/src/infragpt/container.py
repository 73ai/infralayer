"""
Docker container execution module for InfraGPT CLI agent.

This module provides isolated command execution in Docker containers with:
- Real-time streaming output
- Working directory tracking
- Container lifecycle management
"""

import os
import platform
import shlex
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Tuple, Dict, Optional

import docker
from docker import DockerClient
from docker.errors import DockerException, APIError as DockerAPIError
from docker.models.containers import Container
from rich.console import Console

from infragpt.api_client import GKEClusterInfo
from infragpt.exceptions import ContainerSetupError

console = Console()

CONTAINER_NAME = "infragpt-sandbox"
CONTAINER_STOP_TIMEOUT = 5
CWD_MARKER = "__INFRAGPT_CWD__"


def get_sandbox_image() -> str:
    """Get the full sandbox image name for current platform."""
    machine = platform.machine().lower()
    if machine in ("arm64", "aarch64"):
        arch = "arm64"
    else:
        arch = "amd64"
    return f"ghcr.io/73ai/infragpt-sandbox:latest-{arch}"


class DockerNotAvailableError(Exception):
    """Raised when Docker is not installed or running."""

    pass


class ExecutorInterface(ABC):
    """Abstract interface for command executors."""

    @abstractmethod
    def execute_command(self, command: str) -> Tuple[int, str, bool]:
        """
        Execute a command and return results.

        Args:
            command: Shell command to execute

        Returns:
            Tuple of (exit_code, output, was_cancelled)
        """
        pass

    @abstractmethod
    def cleanup(self) -> None:
        """Clean up resources."""
        pass


def is_sandbox_mode() -> bool:
    """
    Check if sandbox mode is enabled.

    Sandbox mode is enabled by default. Set INFRAGPT_ISOLATED=false to disable.
    """
    return os.environ.get("INFRAGPT_ISOLATED", "").lower() != "false"


def ensure_docker_available() -> None:
    """Ensure Docker daemon is available and running. Raises DockerNotAvailableError if not."""
    try:
        client = docker.from_env()
        client.ping()
        client.close()
    except DockerException as e:
        raise DockerNotAvailableError(f"Docker error: {e}")


# Module-level executor singleton
_executor: Optional["ContainerRunner"] = None


def cleanup_old_containers() -> int:
    """Remove any existing sandbox containers from previous CLI sessions."""
    client: Optional[DockerClient] = None
    try:
        client = docker.from_env()
        image_prefix = "ghcr.io/73ai/infragpt-sandbox:"
        containers = client.containers.list(all=True)
        removed = 0
        for container in containers:
            is_sandbox = container.name == CONTAINER_NAME or (
                container.image.tags
                and any(tag.startswith(image_prefix) for tag in container.image.tags)
            )
            if is_sandbox:
                try:
                    container.stop(timeout=CONTAINER_STOP_TIMEOUT)
                except DockerAPIError:
                    pass
                try:
                    container.remove(force=True)
                except DockerAPIError:
                    pass
                removed += 1
        return removed
    except DockerException:
        return 0
    finally:
        if client is not None:
            client.close()


def get_executor(
    gcp_credentials_path: Optional[Path] = None,
    gke_cluster_info: Optional[GKEClusterInfo] = None,
) -> "ContainerRunner":
    """Get or create the ContainerRunner singleton."""
    global _executor
    if _executor is None:
        _executor = ContainerRunner(
            gcp_credentials_path=gcp_credentials_path,
            gke_cluster_info=gke_cluster_info,
        )
    return _executor


def cleanup_executor() -> None:
    """Clean up the executor and remove container."""
    global _executor
    if _executor is not None:
        _executor.cleanup()
        _executor = None


class ContainerRunner(ExecutorInterface):
    """Docker container executor with streaming support."""

    client: Optional[DockerClient]
    container: Optional[Container]

    def __init__(
        self,
        image: Optional[str] = None,
        workdir: str = "/workspace",
        env: Optional[Dict[str, str]] = None,
        volumes: Optional[Dict[str, Dict[str, str]]] = None,
        timeout: int = 60,
        gcp_credentials_path: Optional[Path] = None,
        gke_cluster_info: Optional[GKEClusterInfo] = None,
    ):
        """
        Initialize container runner.

        Args:
            image: Docker image to use
            workdir: Working directory inside container
            env: Additional environment variables
            volumes: Additional volume mounts {host_path: {"bind": container_path, "mode": "rw"}}
            timeout: Command timeout in seconds
            gcp_credentials_path: Path to GCP service account JSON file to mount
            gke_cluster_info: GKE cluster info for kubectl configuration
        """
        self.image = image or get_sandbox_image()
        self.workdir = workdir
        self.user_env = env or {}
        self.user_volumes = volumes or {}
        self.timeout = timeout
        self.gcp_credentials_path = gcp_credentials_path
        self.gke_cluster_info = gke_cluster_info

        self.client = None
        self.container = None
        self.current_cwd = workdir
        self.cancelled = False

    def start(self) -> None:
        """Create and start the container."""
        ensure_docker_available()

        self.client = docker.from_env()

        try:
            self.client.images.pull(self.image)
        except DockerAPIError as e:
            if not self.client.images.list(name=self.image):
                raise DockerNotAvailableError(
                    f"Failed to pull sandbox image: {e}\nRun: docker pull {self.image}"
                )

        mounts = {os.getcwd(): {"bind": "/workspace", "mode": "rw"}}
        mounts.update(self.user_volumes)

        # Build environment variables
        env = dict(self.user_env)

        # Mount GCP credentials if available
        if self.gcp_credentials_path and self.gcp_credentials_path.exists():
            mounts[str(self.gcp_credentials_path)] = {
                "bind": "/credentials/gcp_sa.json",
                "mode": "ro",
            }
            env["GOOGLE_APPLICATION_CREDENTIALS"] = "/credentials/gcp_sa.json"

        self.container = self.client.containers.run(
            self.image,
            command="tail -f /dev/null",
            name=CONTAINER_NAME,
            detach=True,
            tty=True,
            working_dir=self.workdir,
            volumes=mounts,
            environment=env,
            remove=True,
        )

        # Configure GCP tools if credentials are mounted
        if self.gcp_credentials_path and self.gcp_credentials_path.exists():
            try:
                self._configure_gcp_tools()
            except RuntimeError as e:
                raise ContainerSetupError(f"GCP configuration failed: {e}") from e

    def execute_command(self, command: str) -> Tuple[int, str, bool]:
        """
        Execute a command in the container with streaming output.

        Args:
            command: Shell command to execute

        Returns:
            Tuple of (exit_code, output, was_cancelled)
        """
        if self.container is None or self.client is None:
            raise DockerNotAvailableError("Container not started. Call start() first.")

        self.cancelled = False

        console.print(f"[bold cyan]Executing:[/bold cyan] {command}")
        console.print("[dim]Press Ctrl+C to cancel...[/dim]\n")

        try:
            timeout_prefix = f"timeout {self.timeout} " if self.timeout > 0 else ""
            full_command = (
                f"cd {shlex.quote(self.current_cwd)} 2>/dev/null || cd /workspace; "
                f"{timeout_prefix}{command}; _exit_code=$?; echo {CWD_MARKER}; pwd; exit $_exit_code"
            )

            exec_id = self.client.api.exec_create(
                container=self.container.id,
                cmd=["/bin/sh", "-c", full_command],
                tty=True,
                stdout=True,
                stderr=True,
            )

            output_chunks = []
            try:
                for chunk in self.client.api.exec_start(exec_id, stream=True):
                    decoded = chunk.decode("utf-8", errors="replace")
                    output_chunks.append(decoded)
                    if CWD_MARKER not in decoded:
                        console.print(decoded, end="")
                    console.file.flush()
            except KeyboardInterrupt:
                self.cancelled = True
                console.print("\n[yellow]Command cancelled by user[/yellow]")
                try:
                    self.client.api.exec_start(
                        self.client.api.exec_create(
                            container=self.container.id,
                            cmd=["/bin/sh", "-c", "pkill -P 1"],
                        )
                    )
                except DockerAPIError:
                    pass

            exec_info = self.client.api.exec_inspect(exec_id)
            exit_code = exec_info.get("ExitCode", -1) if not self.cancelled else -1
            output = "".join(output_chunks)

            self._update_cwd_from_output(output, CWD_MARKER)

            return exit_code, output, self.cancelled

        except DockerAPIError as e:
            console.print(f"[bold red]Error executing command:[/bold red] {e}")
            return -1, str(e), False

    def _update_cwd_from_output(self, output: str, marker: str) -> None:
        """Extract working directory from command output using the marker."""
        try:
            if marker in output:
                lines = output.split(marker)
                if len(lines) > 1:
                    pwd_output = lines[-1].strip().split("\n")[0].strip()
                    if pwd_output and pwd_output.startswith("/"):
                        self.current_cwd = pwd_output
        except (ValueError, IndexError):
            pass

    def _exec_in_container(self, command: str) -> tuple[int, str, str]:
        """Execute a command in container and return (exit_code, stdout, stderr)."""
        if self.container is None or self.client is None:
            raise RuntimeError("Container not running")

        exec_id = self.client.api.exec_create(
            container=self.container.id,
            cmd=["/bin/sh", "-c", command],
            tty=False,
            stdout=True,
            stderr=True,
        )
        output = self.client.api.exec_start(exec_id, stream=False, demux=True)
        exit_code = self.client.api.exec_inspect(exec_id)["ExitCode"]

        stdout = output[0].decode().strip() if output[0] else ""
        stderr = output[1].decode().strip() if output[1] else ""

        return exit_code, stdout, stderr

    def _configure_gcp_tools(self) -> None:
        """Configure gcloud and kubectl with injected credentials."""
        if self.container is None:
            return

        # Step 1: Activate service account
        exit_code, _, stderr = self._exec_in_container(
            "gcloud auth activate-service-account --key-file=/credentials/gcp_sa.json"
        )
        if exit_code != 0:
            raise RuntimeError(f"Failed to activate service account: {stderr}")

        # Step 2: Get project ID - use provided info or discover via gcloud
        project_id = None
        if self.gke_cluster_info and self.gke_cluster_info.project_id:
            project_id = self.gke_cluster_info.project_id
        else:
            exit_code, stdout, stderr = self._exec_in_container(
                'gcloud projects list --format="value(projectId)" --limit=1'
            )
            if exit_code != 0 or not stdout:
                raise RuntimeError(f"Failed to list projects: {stderr}")
            project_id = stdout.strip()

        # Step 3: Set project
        exit_code, _, stderr = self._exec_in_container(
            f"gcloud config set project {project_id}"
        )
        if exit_code != 0:
            raise RuntimeError(f"Failed to set project: {stderr}")

        # Step 4: Get cluster name and location - use provided info or discover via gcloud
        cluster_name = None
        location = None

        if self.gke_cluster_info:
            if self.gke_cluster_info.cluster_name:
                cluster_name = self.gke_cluster_info.cluster_name
            if self.gke_cluster_info.region:
                location = self.gke_cluster_info.region
            elif self.gke_cluster_info.zone:
                location = self.gke_cluster_info.zone

        # Fall back to discovery if any required field is missing
        if not cluster_name or not location:
            exit_code, stdout, stderr = self._exec_in_container(
                'gcloud container clusters list --format="value(name,location)" --limit=1'
            )
            if exit_code != 0 or not stdout:
                raise RuntimeError(f"Failed to list clusters: {stderr}")

            parts = stdout.strip().split()
            if len(parts) < 2:
                raise RuntimeError(f"Invalid cluster list output: {stdout}")

            if not cluster_name:
                cluster_name = parts[0]
            if not location:
                location = parts[1]

        # Step 5: Get cluster credentials
        exit_code, _, stderr = self._exec_in_container(
            f"gcloud container clusters get-credentials {cluster_name} --region {location} --project {project_id}"
        )
        if exit_code != 0:
            raise RuntimeError(f"Failed to get cluster credentials: {stderr}")

    def stop(self) -> None:
        """Stop and remove the container."""
        if self.container is not None:
            try:
                console.print("[dim]Stopping sandbox container...[/dim]")
                self.container.stop(timeout=CONTAINER_STOP_TIMEOUT)
            except DockerAPIError:
                try:
                    self.container.kill()
                except DockerAPIError:
                    pass
            self.container = None

        if self.client is not None:
            self.client.close()
            self.client = None

    def cleanup(self) -> None:
        """Alias for stop() - implements ExecutorInterface."""
        self.stop()
