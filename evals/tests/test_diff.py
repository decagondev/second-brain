"""sb eval diff (SB-1507): regressions between two releases' stored results."""
import unittest

from evals.gate import diff


def res(suite, failures, cases, support=None):
    n = len(cases)
    return {"suite": suite, "failures": failures, "pass_rate": (n - failures) / n if n else 0.0, "support": support,
            "cases": [{"id": cid, "pass": ok} for cid, ok in cases]}


class DiffTest(unittest.TestCase):
    gate = {"permission": {"failures_max": 0}, "citation": {"score_min": 0.9, "drop_max": 0.02}}

    def test_no_change(self):
        a = {"permission": res("permission", 0, [("p1", True)])}
        self.assertEqual([x[1] for x in diff(self.gate, a, a)], [False])

    def test_newly_failing_case_and_gate(self):
        a = {"permission": res("permission", 0, [("p1", True), ("p2", True)])}
        b = {"permission": res("permission", 1, [("p1", True), ("p2", False)])}
        (suite, bad, detail), = diff(self.gate, a, b)
        self.assertTrue(bad)
        self.assertIn("newly failing: p2", detail)
        self.assertIn("misses the gate", detail)

    def test_score_drop(self):
        a = {"citation": res("citation", 0, [("c1", True)], support=0.97)}
        b = {"citation": res("citation", 0, [("c1", True)], support=0.94)}
        (_, bad, detail), = diff(self.gate, a, b)
        self.assertTrue(bad)
        self.assertIn("dropped 0.030", detail)

    def test_missing_in_b_is_a_regression(self):
        a = {"permission": res("permission", 0, [("p1", True)])}
        self.assertTrue(diff(self.gate, a, {})[0][1])

    def test_only_one_suite(self):
        a = {"permission": res("permission", 0, [("p1", True)]), "citation": res("citation", 0, [], 0.95)}
        self.assertEqual([x[0] for x in diff(self.gate, a, a, "citation")], ["citation"])


if __name__ == "__main__":
    unittest.main()
