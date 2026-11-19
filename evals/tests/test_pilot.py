import json
import tempfile
import unittest
from pathlib import Path

from evals.pilot import main, passes, summarise


def line(**kw):
    base = {"trace": "tr_1", "release": "rel_a", "rating": "up", "at": "2026-11-16T09:00:00Z", "claims": 2, "uncited": 0, "refused": 0}
    base.update(kw)
    return json.dumps(base)


class Pilot(unittest.TestCase):
    def test_summary(self):
        s = summarise([line(), line(rating="down", wrong_citation="docs:a@1", uncited=1), line(rating="down", wrong_citation="docs:a@1", refused=2), line(), ""])
        self.assertEqual(s["feedback"], 4)
        self.assertEqual(s["thumbs_up_rate"], 0.5)
        self.assertEqual(s["wrong_citations"], {"docs:a@1": 2})
        self.assertEqual(s["answers_with_uncited"], 0.25)
        self.assertEqual(s["answers_with_refused"], 0.25)
        self.assertFalse(passes(s))

    def test_gate(self):
        self.assertTrue(passes(summarise([line()] * 7 + [line(rating="down")] * 3)))
        self.assertFalse(passes(summarise([])))  # no feedback is not a pass

    def test_cli_exit_codes(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "export.jsonl"
            p.write_text("\n".join([line()] * 8 + [line(rating="down")] * 2) + "\n")
            self.assertEqual(main(["report", str(p)]), 0)
            p.write_text(line(rating="down") + "\n")
            self.assertEqual(main(["report", str(p)]), 1)
        self.assertEqual(main([]), 2)
