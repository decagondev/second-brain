import unittest
from pathlib import Path

from evals.checks import evaluate
from evals.runner import load, run_suite
from evals.targets import MockTarget

HERE = Path(__file__).resolve().parents[1]


class Checks(unittest.TestCase):
    def test_each_check_passes_and_fails(self):
        res = {"cites": ["a@1"], "uncited": 2, "claims": [{"cites": ["a@1"], "support": 0.8}],
               "notice": "no_access", "tool_calls": 1, "text": "hello CANARY", "tool_args": []}
        cases = {
            "cites_any": (["a@1"], ["b@1"]), "cites_none": (["b@1"], ["a@1"]), "uncited_max": (2, 1),
            "faithful_min": (0.8, 0.9), "refuses": (True, False), "tool_calls": (1, 0),
            "not_contains": (["NOPE"], ["CANARY"]),
        }
        for name, (good, bad) in cases.items():
            self.assertTrue(evaluate(res, {name: good})[0][1], name + " should pass")
            self.assertFalse(evaluate(res, {name: bad})[0][1], name + " should fail")

    def test_unknown_expectation_fails(self):
        self.assertFalse(evaluate({}, {"cites_maybe": 1})[0][1])

    def test_no_cited_claims_is_not_faithful(self):
        self.assertFalse(evaluate({"claims": []}, {"faithful_min": 0.5})[0][1])

    def test_canary_in_tool_args_is_caught(self):
        self.assertFalse(evaluate({"text": "", "tool_args": [{"q": "CANARY-7f3a"}]}, {"not_contains": ["CANARY-7f3a"]})[0][1])


class Runner(unittest.TestCase):
    def test_smoke_suite_passes(self):
        suite = load(HERE / "suites/smoke.yaml")
        rows, _ = run_suite(suite, MockTarget(HERE / suite["world"]))
        self.assertEqual([ok for _, ok, _ in rows], [True, True, True])

    def test_worst_of_three_counts(self):
        class Flaky:
            def run(self, case, attempt):
                return {"cites": ["a@1"] if attempt != 2 else []}
        rows, _ = run_suite({"cases": [{"id": "x", "expect": {"cites_any": ["a@1"]}}]}, Flaky())
        self.assertFalse(rows[0][1], "one bad run out of three must fail the case")


if __name__ == "__main__":
    unittest.main()


class Faithfulness(unittest.TestCase):
    def test_scorer(self):
        from evals.faithfulness import support
        src = "Refunds over 500 need a team lead's approval."
        self.assertEqual(support("Refunds over 500 need a team lead's approval.", src), 1.0)
        self.assertEqual(support("Refunds over 750 need approval.", src), 0.0)  # a number the source doesn't say
        self.assertLess(support("Shipments arrive within five working days.", src), 0.2)
        self.assertEqual(support("It is.", src), 1.0)  # no content words, no numbers
        self.assertEqual(support("", ""), 1.0)

    def test_given_support_wins(self):
        from evals.faithfulness import claim_support
        self.assertEqual(claim_support({"text": "x", "cited_text": "", "support": 0.4}), 0.4)

    def test_faithful_min_computes_when_missing(self):
        res = {"claims": [{"text": "Refunds over 500 need approval.", "cites": ["d@1"], "cited_text": "Refunds over 500 need a team lead's approval."}]}
        self.assertTrue(evaluate(res, {"faithful_min": 0.9})[0][1])
        res["claims"][0]["cited_text"] = "Shipping takes 5 days."
        self.assertFalse(evaluate(res, {"faithful_min": 0.9})[0][1])


class TenantSuites(unittest.TestCase):
    def test_loads_and_names(self):
        from evals.runner import tenant_suites
        suites = tenant_suites(HERE.parent / "tenants" / "acme-logistics", 0.92)
        self.assertEqual({s["suite"] for s in suites}, {"tenant/acme-logistics"})

    def test_looser_gate_is_an_error(self):
        import tempfile
        from evals.runner import tenant_suites
        with tempfile.TemporaryDirectory() as d:
            t = Path(d) / "beta-freight"
            (t / "evals").mkdir(parents=True)
            (t / "evals" / "golden.yaml").write_text("suite: g\nworld: .\ngate: { score_min: 0.5 }\ncases: []\n")
            with self.assertRaises(ValueError):
                tenant_suites(t, 0.92)
