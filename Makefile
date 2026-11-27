# Local validation interface. Python owns selection, execution and diagnostics.
PYTHON ?= python3
MODULE ?= all
.DEFAULT_GOAL := help

.PHONY: all help setup setup-network fmt-check lint vet test pre-add-check \
	check-records markdownlint docs-check inventory-check gate-selftest lint-selftest \
	generate-check build-check _local-checks

all help setup setup-network fmt-check lint vet test \
check-records markdownlint docs-check inventory-check gate-selftest lint-selftest \
generate-check build-check: _local-checks

# Literal entry is required by the installed pre-add hook's discovery contract.
pre-add-check: _local-checks

# One prerequisite per Make invocation coalesces combined public goals.
_local-checks:
	$(PYTHON) -B scripts/make_tasks.py --module $(MODULE) $(MAKECMDGOALS)
