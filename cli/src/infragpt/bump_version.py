#!/usr/bin/env python3

import re
import argparse
import subprocess
from pathlib import Path


def get_current_version() -> str:
    """Get the current version from __init__.py."""
    init_file = Path("src/infragpt/__init__.py")
    if not init_file.exists():
        raise FileNotFoundError(f"Could not find {init_file}")

    with open(init_file, "r") as f:
        content = f.read()

    match = re.search(r'__version__\s*=\s*["\']([^"\']+)["\']', content)
    if not match:
        raise ValueError("Could not find version string in src/infragpt/__init__.py")

    return match.group(1)


def update_version(new_version: str) -> None:
    """Update version in __init__.py and pyproject.toml."""
    init_file = Path("src/infragpt/__init__.py")
    with open(init_file, "r") as f:
        content = f.read()

    new_content = re.sub(
        r'__version__\s*=\s*["\']([^"\']+)["\']',
        f'__version__ = "{new_version}"',
        content,
    )

    with open(init_file, "w") as f:
        f.write(new_content)

    # Update version in pyproject.toml
    pyproject_file = Path("pyproject.toml")
    with open(pyproject_file, "r") as f:
        content = f.read()

    new_content = re.sub(r'version = "[^"]+"', f'version = "{new_version}"', content)

    with open(pyproject_file, "w") as f:
        f.write(new_content)

    print(f"Updated version to {new_version}")


def commit_and_tag(version: str) -> None:
    """Commit version bump and create a git tag."""
    subprocess.run(
        ["git", "add", "src/infragpt/__init__.py", "pyproject.toml"], check=True
    )
    subprocess.run(["git", "commit", "-m", f"Bump version to {version}"], check=True)

    # Create tag
    tag_name = f"v{version}"
    subprocess.run(
        ["git", "tag", "-a", tag_name, "-m", f"Version {version}"], check=True
    )

    print(f"Created commit and tag {tag_name}")
    print("To push the changes:")
    print(f"  git push origin master && git push origin {tag_name}")


def bump_version(part: str = "patch") -> str:
    """Calculate the new version based on the bump type."""
    current = get_current_version()
    major, minor, patch = map(int, current.split("."))

    if part == "major":
        new_version = f"{major + 1}.0.0"
    elif part == "minor":
        new_version = f"{major}.{minor + 1}.0"
    elif part == "patch":
        new_version = f"{major}.{minor}.{patch + 1}"
    else:
        new_version = part  # Custom version

    return new_version


def main() -> None:
    """Main entry point for the version bump CLI."""
    parser = argparse.ArgumentParser(description="Bump InfraGPT version")
    parser.add_argument(
        "part",
        nargs="?",
        choices=["major", "minor", "patch"],
        default="patch",
        help="Part of the version to bump (default: patch)",
    )
    parser.add_argument(
        "--version", "-v", help="Specify a custom version (overrides part)"
    )
    parser.add_argument(
        "--commit", "-c", action="store_true", help="Commit and tag the version bump"
    )

    args = parser.parse_args()

    # Get the new version
    if args.version:
        new_version = args.version
    else:
        new_version = bump_version(args.part)

    # Show the version change
    current = get_current_version()
    print(f"Current version: {current}")
    print(f"New version: {new_version}")

    # Confirm
    if input("Do you want to proceed? [y/N] ").lower() != "y":
        print("Aborted")
        return

    # Update files
    update_version(new_version)

    # Commit and tag if requested
    if args.commit:
        commit_and_tag(new_version)


if __name__ == "__main__":
    main()
