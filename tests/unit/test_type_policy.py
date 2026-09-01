import ast
from pathlib import Path

import pytest


@pytest.mark.unit
def test_object_is_not_used_in_type_annotations() -> None:
    project_root = Path(__file__).parents[2]
    python_files = tuple(
        path
        for source_root in (project_root / "backend", project_root / "tests")
        for path in source_root.rglob("*.py")
    )

    violations: list[str] = []
    for path in python_files:
        tree = ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
        for node in ast.walk(tree):
            annotations = []
            if isinstance(node, (ast.AnnAssign, ast.FunctionDef, ast.AsyncFunctionDef)):
                if isinstance(node, ast.AnnAssign):
                    annotations.append(node.annotation)
                else:
                    annotations.extend(
                        argument.annotation
                        for argument in (*node.args.posonlyargs, *node.args.args, *node.args.kwonlyargs)
                        if argument.annotation is not None
                    )
                    if node.args.vararg and node.args.vararg.annotation is not None:
                        annotations.append(node.args.vararg.annotation)
                    if node.args.kwarg and node.args.kwarg.annotation is not None:
                        annotations.append(node.args.kwarg.annotation)
                    if node.returns is not None:
                        annotations.append(node.returns)
            if any(isinstance(annotation, ast.AST) and any(
                isinstance(child, ast.Name) and child.id == "object" for child in ast.walk(annotation)
            ) for annotation in annotations):
                violations.append(f"{path}:{node.lineno}")

    assert not violations, "Forbidden object type annotations: " + ", ".join(violations)
