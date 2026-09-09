#!/usr/bin/env python3
"""Собирает единый OpenAPI-документ из контрактов сервисов."""

from pathlib import Path
from typing import Any

import yaml


ROOT = Path(__file__).resolve().parents[1]
SPEC_FILES = (
    "identity",
    "analytics",
    "profiles",
    "social",
    "content",
    "messaging",
    "media",
)


def main() -> None:
    merged = {
        "openapi": "3.0.3",
        "info": {"title": "General Project API", "version": "1.0.0"},
        "paths": {},
        "components": {},
    }
    for service in SPEC_FILES:
        source = ROOT / "api" / "openapi" / service / "openapi.yaml"
        document = yaml.safe_load(source.read_text())
        operation_templates = document.get("components", {}).get("pathOperations", {})
        merged["paths"].update(_resolve_templates(document.get("paths", {}), operation_templates))
        for section, values in document.get("components", {}).items():
            if section == "pathOperations":
                continue
            merged["components"].setdefault(section, {}).update(values or {})
    output = ROOT / "services" / "gateway" / "docs" / "swagger.yaml"
    output.write_text(yaml.safe_dump(merged, allow_unicode=True, sort_keys=False))
    json_output = output.with_suffix(".json")
    json_output.write_text(__import__("json").dumps(merged, ensure_ascii=False, indent=2) + "\n")


def _resolve_templates(value: Any, templates: dict[str, Any]) -> Any:
    if isinstance(value, dict):
        reference = value.get("$ref")
        if reference == "#/components/pathOperations/NoContentOperation":
            return templates["NoContentOperation"]
        return {key: _resolve_templates(item, templates) for key, item in value.items()}
    if isinstance(value, list):
        return [_resolve_templates(item, templates) for item in value]
    return value


if __name__ == "__main__":
    main()
