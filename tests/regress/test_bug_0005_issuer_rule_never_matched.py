"""BUG-0005: the SB-108 issuer rule in tools/check/deps.py contained a literal backspace
instead of the regex `\\b`, so it never matched and any package could have minted scopes.
The rule must flag a call outside internal/auth, and must allow tests and internal/auth."""
import importlib.util
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("deps", ROOT / "tools/check/deps.py")
deps = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deps)

CALL = 'package x\n\nimport "example.com/second-brain/internal/core/tenancy"\n\nvar _ = tenancy.NewIssuer("t")\n'


class IssuerRule(unittest.TestCase):
    def tree(self, files):
        d = Path(tempfile.mkdtemp())
        for rel, body in files.items():
            (d / rel).parent.mkdir(parents=True, exist_ok=True)
            (d / rel).write_text(body)
        return d

    def test_call_outside_auth_is_flagged(self):
        d = self.tree({"internal/cli/x.go": CALL})
        self.assertEqual(len(deps.issuer_problems(d)), 1)

    def test_auth_and_tests_are_allowed(self):
        d = self.tree({"internal/auth/a.go": CALL, "internal/cli/x_test.go": CALL})
        self.assertEqual(deps.issuer_problems(d), [])

    def test_source_has_no_control_characters(self):
        src = (ROOT / "tools/check/deps.py").read_text()
        self.assertFalse(any(ord(c) < 32 and c not in "\n\t" for c in src))


if __name__ == "__main__":
    unittest.main()


class NoInvisibleCharacters(unittest.TestCase):
    """Same root cause as BUG-0005 (a tool silently rewrote an escape): no source file may
    contain control characters or invisible format characters (Unicode Cf). Spell them as
    code points in code, e.g. rune(0x200B)."""

    def test_sources_are_visible(self):
        import unicodedata
        bad = []
        for pat in ("**/*.go", "**/*.py", "**/*.md", "**/*.yaml", "**/*.json", "**/*.cedar"):
            for f in ROOT.glob(pat):
                rel = f.relative_to(ROOT).as_posix()
                if rel.startswith(("vendor/", "site/", "tools/vendor/")) or "node_modules" in rel:
                    continue
                for n, line in enumerate(f.read_text(encoding="utf-8", errors="replace").splitlines(), 1):
                    if any((ord(c) < 32 and c not in "\t") or unicodedata.category(c) == "Cf" for c in line):
                        bad.append(f"{rel}:{n}")
        self.assertEqual(bad, [], "invisible characters in source")
