import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("check_pnpm_audit_exceptions.py")


class PnpmAuditExceptionCheckerTests(unittest.TestCase):
    def run_checker(self, data):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "audit.json"
            exceptions = Path(directory) / "exceptions.yml"
            report.write_text(json.dumps(data), encoding="utf-8")
            exceptions.write_text("version: 1\nexceptions:\n", encoding="utf-8")
            return subprocess.run(
                [sys.executable, str(SCRIPT), "--audit", str(report), "--exceptions", str(exceptions)],
                check=False,
                capture_output=True,
                text=True,
            )

    def test_accepts_clean_audit_reports(self):
        for key in ("advisories", "vulnerabilities"):
            with self.subTest(key=key):
                self.assertEqual(self.run_checker({key: {}}).returncode, 0)

    def test_rejects_registry_errors_even_with_empty_results(self):
        result = self.run_checker({"error": {"code": "ECONNRESET"}, "advisories": {}})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Audit failed", result.stderr)

    def test_rejects_missing_or_malformed_results(self):
        for data in ({}, {"metadata": {}}, {"advisories": []}, []):
            with self.subTest(data=data):
                self.assertNotEqual(self.run_checker(data).returncode, 0)

    def test_rejects_unreviewed_high_and_critical_dev_dependencies(self):
        for severity in ("high", "critical"):
            with self.subTest(severity=severity):
                result = self.run_checker({"advisories": {"1": {
                    "module_name": "dev-tool",
                    "github_advisory_id": "GHSA-test-test-test",
                    "severity": severity,
                    "findings": [{"dev": True}],
                }}})
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("dev-tool", result.stderr)


if __name__ == "__main__":
    unittest.main()
