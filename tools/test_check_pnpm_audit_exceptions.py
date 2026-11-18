import json
import subprocess
import sys
import tempfile
import unittest
from datetime import date, timedelta
from pathlib import Path


SCRIPT = Path(__file__).with_name("check_pnpm_audit_exceptions.py")


class PnpmAuditExceptionCheckerTests(unittest.TestCase):
    def run_checker(self, data, *, exceptions_text=None, production=None):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "audit.json"
            exceptions = Path(directory) / "exceptions.yml"
            report.write_text(json.dumps(data), encoding="utf-8")
            exceptions.write_text(exceptions_text or "version: 1\nexceptions:\n", encoding="utf-8")
            command = [sys.executable, str(SCRIPT), "--audit", str(report), "--exceptions", str(exceptions)]
            if production is not None:
                production_report = Path(directory) / "audit-production.json"
                production_report.write_text(json.dumps(production), encoding="utf-8")
                command.extend(["--production-audit", str(production_report)])
            return subprocess.run(
                command,
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

    @staticmethod
    def dev_report():
        return {"advisories": {"1": {
            "module_name": "picomatch",
            "github_advisory_id": "GHSA-fc5q-39xc-vjpf",
            "severity": "high",
        }}}

    @staticmethod
    def dev_exception(*, scope="dev-only", expired=False):
        expiry = date.today() + timedelta(days=-1 if expired else 1)
        return f"""version: 1
exceptions:
  - package: picomatch
    advisory: GHSA-fc5q-39xc-vjpf
    severity: high
    scope: {scope}
    mitigation: Independently validate absence from production dependencies
    expires_on: {expiry.isoformat()}
"""

    def test_dev_only_exception_requires_independent_production_report(self):
        result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception())
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("requires --production-audit", result.stderr)

    def test_accepts_dev_only_exception_when_production_report_excludes_advisory(self):
        for key in ("advisories", "vulnerabilities"):
            with self.subTest(key=key):
                result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception(), production={key: {}})
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_dev_only_exception_rejects_advisory_reaching_production(self):
        for report in (self.dev_report(), {"vulnerabilities": {"picomatch": {
            "severity": "high", "via": [{"github_advisory_id": "GHSA-fc5q-39xc-vjpf"}],
        }}}):
            with self.subTest(report=report):
                result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception(), production=report)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("cannot allow a production vulnerability", result.stderr)

    def test_rejects_failed_or_malformed_production_reports(self):
        for report in ({"error": {"code": "ECONNRESET"}, "advisories": {}}, {}, []):
            with self.subTest(report=report):
                result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception(), production=report)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("Production audit failed", result.stderr)

    def test_dev_only_exception_still_expires(self):
        result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception(expired=True), production={"advisories": {}})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Exceptions expired", result.stderr)

    def test_rejects_unknown_exception_scope(self):
        result = self.run_checker(self.dev_report(), exceptions_text=self.dev_exception(scope="typo"), production={"advisories": {}})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid scope", result.stderr)


if __name__ == "__main__":
    unittest.main()
