.PHONY: install lint format typecheck check test clean

install:
	uv sync

lint:
	uv run ruff check src/

format:
	uv run ruff format src/
	uv run ruff check --fix src/

typecheck:
	uv run pyright src/

check: lint typecheck

test:
	uv run pytest tests/

clean:
	rm -rf .pytest_cache .ruff_cache .mypy_cache __pycache__
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete 2>/dev/null || true
