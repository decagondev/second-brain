"""Targets an eval runs against. W0 has only the mock target."""
from pathlib import Path

import yaml


class MockTarget:
    """Answers from canned responses in fixtures/<world>/responses.yaml.

    A response may be one result or a list of results, one per run, so the
    three-runs-worst-counts rule can be exercised deterministically.
    """

    def __init__(self, world_dir: Path):
        f = world_dir / "responses.yaml"
        if not f.exists():
            raise FileNotFoundError(f"mock target needs {f}")
        self.responses = yaml.safe_load(f.read_text()) or {}

    def run(self, case, attempt):
        r = self.responses.get(case["id"])
        if r is None:
            raise KeyError(f"no mock response for case {case['id']}")
        return r[attempt % len(r)] if isinstance(r, list) else r
