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


class LocalCell:
    """Runs the real ask path in a local cell over the suite's world.yaml (sb-eval(5),
    Worlds for the local cell). tools/evalcell is built once per process; all of a
    suite's cases and runs go to it in one call."""

    _binary = None

    def __init__(self, world_dir: Path, model: str, runs: int):
        import json, subprocess
        f = world_dir / "world.yaml"
        if not f.exists():
            raise FileNotFoundError(f"the local cell needs {f}")
        self.world = yaml.safe_load(f.read_text())
        self.model, self.runs = model, runs
        self.results = None
        self._json, self._subprocess = json, subprocess

    @classmethod
    def binary(cls):
        if cls._binary is None:
            import subprocess, tempfile
            root = Path(__file__).resolve().parents[1]
            out = Path(tempfile.mkdtemp(prefix="evalcell-")) / "evalcell"
            subprocess.run(["go", "build", "-o", str(out), "./tools/evalcell"], cwd=root, check=True)
            cls._binary = out
        return cls._binary

    def prepare(self, cases):
        root = Path(__file__).resolve().parents[1]
        req = {"world": self.world, "model": self.model, "policies": str(root / "policy" / "platform"),
               "cases": [{"key": f"{c['id']}#{a}", "as": c["as"], "ask": c["ask"], "space": c.get("space", ""), "agent": c.get("agent", ""), "inject": {"acl": (c.get("inject") or {}).get("acl", "")}}
                         for c in cases for a in range(self.runs)]}
        p = self._subprocess.run([str(self.binary())], input=self._json.dumps(req), capture_output=True, text=True)
        if p.returncode != 0:
            raise RuntimeError(f"evalcell failed: {p.stderr.strip()}")
        self.results = self._json.loads(p.stdout)["results"]

    def run(self, case, attempt):
        r = self.results[f"{case['id']}#{attempt}"]
        if r.get("error") and "error" not in case.get("expect", {}):
            raise RuntimeError(f"case {case['id']}: {r['error']}")
        return {**r, "claims": r.get("claims") or [], "cites": r.get("cites") or []}
