import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("check_govuln_exceptions.py")
MODULE = "github.com/apernet/hysteria/core/v2"
ADVISORY = "GO-2026-5288"


def write_json_stream(path: Path, *, scan_level="symbol", version="v2.12.2", advisory=ADVISORY, symbol=True):
    messages = [
        {"config": {"scan_level": scan_level}},
        {"SBOM": {"modules": [{"path": MODULE, "version": version}]}},
        {
            "finding": {
                "osv": advisory,
                "trace": [
                    {
                        "module": MODULE,
                        "version": version,
                        **(
                            {"package": f"{MODULE}/client", "function": "NewClient"}
                            if symbol
                            else {}
                        ),
                    }
                ],
            }
        },
    ]
    path.write_text("\n".join(json.dumps(message) for message in messages), encoding="utf-8")


def write_exceptions(path: Path):
    path.write_text(
        json.dumps(
            {
                "version": 1,
                "exceptions": [
                    {
                        "id": ADVISORY,
                        "module": MODULE,
                        "minimum_safe_version": "v2.8.2",
                        "reason": "reviewed metadata mismatch",
                        "mitigation": "client-only use on a fixed version",
                        "reference": "https://example.com/advisory",
                        "expires_on": "2099-01-01",
                        "owner": "test",
                    }
                ],
            }
        ),
        encoding="utf-8",
    )


class GovulnExceptionCheckerTests(unittest.TestCase):
    def run_checker(self, report: Path, exceptions: Path):
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT),
                "--report",
                str(report),
                "--exceptions",
                str(exceptions),
            ],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_accepts_reviewed_exception_on_safe_version(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            exceptions = Path(directory) / "exceptions.json"
            write_json_stream(report)
            write_exceptions(exceptions)

            result = self.run_checker(report, exceptions)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("1 approved exception", result.stdout)

    def test_rejects_unapproved_reachable_vulnerability(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            exceptions = Path(directory) / "exceptions.json"
            write_json_stream(report, advisory="GO-2099-0001")
            write_exceptions(exceptions)

            result = self.run_checker(report, exceptions)

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("missing an approved exception", result.stderr)

    def test_rejects_module_below_minimum_safe_version(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            exceptions = Path(directory) / "exceptions.json"
            write_json_stream(report, version="v2.8.1")
            write_exceptions(exceptions)

            result = self.run_checker(report, exceptions)

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("requires", result.stderr)

    def test_rejects_non_symbol_scan(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            exceptions = Path(directory) / "exceptions.json"
            write_json_stream(report, scan_level="module")
            write_exceptions(exceptions)

            result = self.run_checker(report, exceptions)

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("must use symbol scan level", result.stderr)

    def test_ignores_module_only_findings(self):
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            exceptions = Path(directory) / "exceptions.json"
            write_json_stream(report, symbol=False)
            write_exceptions(exceptions)

            result = self.run_checker(report, exceptions)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("0 approved exception", result.stdout)


if __name__ == "__main__":
    unittest.main()
