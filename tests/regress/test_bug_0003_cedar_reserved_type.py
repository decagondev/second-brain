"""BUG-0003: policy examples used Cedar's reserved `Action` type as a resource and an
unguarded optional attribute. Every policy file and every cedar block in the docs must
validate against policy/schema.cedarschema, and the original mistakes must stay rejected."""
import re
import unittest
from pathlib import Path

try:
    import cedarpy
except ImportError as e:  # a missing validator is a failure, not a skip (unknown means no)
    raise SystemExit("BUG-0003 regression needs cedarpy: pip install cedarpy") from e

ROOT = Path(__file__).resolve().parents[2]
SCHEMA = (ROOT / "policy/schema.cedarschema").read_text()


def blocks():
    for f in sorted(ROOT.glob("**/*.cedar")):
        if "site" not in f.parts:
            yield str(f.relative_to(ROOT)), f.read_text()
    for f in sorted(ROOT.glob("**/*.md")):
        if "site" in f.parts:
            continue
        for i, b in enumerate(re.findall(r"```cedar\n(.*?)```", f.read_text(), re.S)):
            yield f"{f.relative_to(ROOT)}#{i}", b


class Bug0003(unittest.TestCase):
    def test_every_policy_validates(self):
        n = 0
        for name, text in blocks():
            r = cedarpy.validate_policies(text, SCHEMA)
            self.assertTrue(r.validation_passed, f"{name}: {[str(e) for e in r.errors]}")
            n += 1
        self.assertGreater(n, 5, "expected to find the repo's policies")

    def test_original_mistakes_stay_rejected(self):
        for bad in ('permit (principal, action == Action::"approve", resource is Action);',
                    'forbid (principal, action == Action::"use_agent", resource) when { context.target_project == "FIN" };'):
            self.assertFalse(cedarpy.validate_policies(bad, SCHEMA).validation_passed, bad)


if __name__ == "__main__":
    unittest.main()
