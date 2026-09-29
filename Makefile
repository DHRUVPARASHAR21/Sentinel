# Sentinel's executable Makefile contract starts in milestone 1.
# Planned targets: fmt, vet, test, race, integration, package, package-smoke, verify.

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Sentinel is in planning mode. See docs/development.md for the planned targets."
