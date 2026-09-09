#!/usr/bin/env python3
import argparse
import json
import re
import sys
from datetime import date


REQUIRED_FIELDS = {
    "id",
    "module",
    "minimum_safe_version",
    "reason",
    "mitigation",
    "reference",
    "expires_on",
    "owner",
}
SEMVER_RE = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$")


def read_json_stream(path: str):
    with open(path, "r", encoding="utf-8-sig") as handle:
        content = handle.read()

    decoder = json.JSONDecoder()
    offset = 0
    while offset < len(content):
        while offset < len(content) and content[offset].isspace():
            offset += 1
        if offset >= len(content):
            break
        value, offset = decoder.raw_decode(content, offset)
        yield value


def parse_semver(value: str) -> tuple[int, int, int] | None:
    match = SEMVER_RE.match(value or "")
    if not match:
        return None
    return tuple(int(part) for part in match.groups())


def parse_date(value: str) -> date | None:
    try:
        return date.fromisoformat(value)
    except (TypeError, ValueError):
        return None


def load_report(path: str):
    modules = {}
    findings = set()
    scan_level = None

    for message in read_json_stream(path):
        config = message.get("config")
        if isinstance(config, dict):
            scan_level = config.get("scan_level")

        sbom = message.get("SBOM")
        if isinstance(sbom, dict):
            for module in sbom.get("modules", []):
                module_path = module.get("path")
                if module_path:
                    modules[module_path] = module.get("version", "")

        finding = message.get("finding")
        if not isinstance(finding, dict):
            continue
        advisory_id = finding.get("osv")
        if not advisory_id:
            continue
        trace = finding.get("trace") or []
        if trace and trace[0].get("module") and trace[0].get("function"):
            # govulncheck orders traces from the vulnerable symbol to callers.
            findings.add((advisory_id, trace[0]["module"]))
        elif not trace:
            findings.add((advisory_id, "<unknown module>"))

    return modules, findings, scan_level


def load_exceptions(path: str):
    with open(path, "r", encoding="utf-8") as handle:
        data = json.load(handle)
    if data.get("version") != 1 or not isinstance(data.get("exceptions"), list):
        raise ValueError("exceptions file must contain version 1 and an exceptions list")
    return data["exceptions"]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--report", required=True)
    parser.add_argument("--exceptions", required=True)
    args = parser.parse_args()

    try:
        modules, findings, scan_level = load_report(args.report)
        exceptions = load_exceptions(args.exceptions)
    except (OSError, ValueError, json.JSONDecodeError) as error:
        sys.stderr.write(f"Unable to validate govulncheck report: {error}\n")
        return 1

    errors = []
    exception_index = {}
    today = date.today()

    if scan_level != "symbol":
        errors.append(
            f"govulncheck report must use symbol scan level, found {scan_level!r}"
        )

    for exception in exceptions:
        missing = sorted(field for field in REQUIRED_FIELDS if not exception.get(field))
        if missing:
            errors.append(f"Exception missing required fields {missing}")
            continue

        key = (exception["id"], exception["module"])
        if key in exception_index:
            errors.append(f"Duplicate exception for {key[0]} in {key[1]}")
            continue

        expires_on = parse_date(exception["expires_on"])
        if expires_on is None:
            errors.append(f"Exception {key[0]} has invalid expires_on date")
            continue
        if expires_on < today:
            errors.append(f"Exception {key[0]} expired on {expires_on.isoformat()}")

        minimum_version = parse_semver(exception["minimum_safe_version"])
        current_version = parse_semver(modules.get(exception["module"], ""))
        if minimum_version is None:
            errors.append(f"Exception {key[0]} has invalid minimum_safe_version")
        elif current_version is None:
            errors.append(
                f"Exception {key[0]} cannot verify installed version for {key[1]}"
            )
        elif current_version < minimum_version:
            errors.append(
                f"Exception {key[0]} requires {key[1]} >= "
                f"{exception['minimum_safe_version']}, found {modules[key[1]]}"
            )

        exception_index[key] = exception

    unapproved = sorted(findings - set(exception_index))
    if unapproved:
        errors.append("Reachable Go vulnerabilities missing an approved exception:")
        errors.extend(f"- {advisory_id} in {module}" for advisory_id, module in unapproved)

    if errors:
        sys.stderr.write("\n".join(errors) + "\n")
        return 1

    matched = sorted(findings & set(exception_index))
    for advisory_id, module in matched:
        print(
            f"Approved temporary exception: {advisory_id} in {module} "
            f"({modules[module]}), expires {exception_index[(advisory_id, module)]['expires_on']}"
        )
    print(f"govulncheck validation passed with {len(matched)} approved exception(s).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
