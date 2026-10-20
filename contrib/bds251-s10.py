#!/usr/bin/env python3
"""Offline fixture, snapshot, digest, and bounded-ledger support for BDS-251 S10.

This helper never calls a model. ``run-root`` only launches the real root CLI
with its normal terminal prompts; it does not synthesize or forward approval
input. See engineering/product/bds-251-stage-4-step10-preparation.md.
"""

from __future__ import annotations

import argparse
import atexit
import base64
from contextlib import contextmanager, redirect_stderr, redirect_stdout
import fcntl
import hashlib
import io
import json
import math
import os
from pathlib import Path
import re
import select
import signal
import shutil
import stat
import subprocess
import sys
import tempfile
import termios
import time
from datetime import datetime, timezone


BASELINE = "20a0e23d65581e9015c6adc0bf048cb867de384c"
ROOT = Path(__file__).resolve().parents[1]
FIXTURES = ROOT / "internal/asset/testdata/adoption"
GREENFIELD = FIXTURES / "greenfield"
PRESERVATION = FIXTURES / "preservation"
APPROVED_OUTPUTS = (
    "docs/README.md",
    "docs/reference/usage.md",
    "engineering/README.md",
    "engineering/product/overview.md",
)
OUTPUT_HASHES = {
    "docs/README.md": "65c911432ba0ac02f08b6c45b59ee197207751f0c45d59e290cfde36b84e791e",
    "docs/reference/usage.md": "47c4100c772b343293decb85b1e87dc530d39a5a01302c50fd0750728cf30cd8",
    "engineering/README.md": "95dfb71d18039a94bf2e0de0a360df0ec5785e80b0ba74bf527e9a5f9ac9a815",
    "engineering/product/overview.md": "e1d4ddf4687edb3a996bcbcad99d253da512a7048058491fbafa494fa7e6df6a",
}
OVERLAYS = {
    "readme": ("preservation/mixed-readme/README.md", "README.md", "11a3a534bb7acc46482a8721f3aa85377e8cda9dbd28bc9c127f37ff6134e9dc"),
    "partial-engineering": (
        "preservation/partial-engineering/engineering/decisions/0001-docs.md",
        "engineering/decisions/0001-docs.md",
        "7b5ba96cd690c4800dcb970dd8a1da589d90bcf856733e56dafc217911f08927",
    ),
    "collision": (
        "preservation/output-collision/docs/reference/usage.md",
        "docs/reference/usage.md",
        "98f2932add4aa1f592944e1b157a4c9c0ac314375ecf060ca9ba14808f5ee78e",
    ),
    "partial-legacy": (
        "preservation/legacy-docs-partial/docs/decisions/0001-adopt-lds.md",
        "docs/decisions/0001-adopt-lds.md",
        "a9b5d4fdc1172bc48fdf67901d40b9271a3a431b1df0d2f776c8bc8c4479c544",
    ),
}
TARGET_CASES = (
    "fresh",
    "partial-engineering",
    "collision",
    "partial-legacy",
    "decline",
    "stale",
    "override",
    "noninteractive",
)
ROOT_CALL_CASES = {"fresh", "rerun", "partial-engineering", "decline", "stale", "override"}
SKILL_CASES = (
    "fresh",
    "rerun",
    "partial-engineering",
    "collision",
    "partial-legacy",
    "decline",
    "stale",
    "override",
    "noninteractive",
)
EXECUTOR_AUTHORIZATION = {
    "rootChatCalls": 6,
    "nativeSkillSessions": 9,
    "logicalTurnsPerNativeSession": 8,
    "logicalRequestsPerWave": 78,
    "wireHTTPRequests": "not a proven cap; client/gateway retries are separately reported",
}
AUTHORIZATION_PROFILES = {
    "executor": {
        "rootCases": ROOT_CALL_CASES,
        "runCases": ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"},
        "verifyCases": ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"},
        "authorization": EXECUTOR_AUTHORIZATION,
        "nativeMustRemainBlocked": False,
    },
    "supplemental-decline": {
        "rootCases": {"decline"},
        "runCases": {"decline"},
        "verifyCases": {"decline"},
        "authorization": {
            "rootChatCalls": 1,
            "nativeSkillSessions": 0,
            "logicalTurnsPerNativeSession": 0,
            "logicalRequestsPerWave": 1,
            "wireHTTPRequests": "not a proven cap; client/gateway retries are separately reported",
        },
        "nativeMustRemainBlocked": True,
    },
}
PLAN_HEADER = "Complete deterministic plan (review every field; no writes have occurred):"


def authorization_profile(name: object) -> dict[str, object]:
    if not isinstance(name, str) or name not in AUTHORIZATION_PROFILES:
        raise ValueError(f"unknown S10 authorization profile: {name!r}")
    return AUTHORIZATION_PROFILES[name]


def manifest_profile_name(manifest: dict[str, object]) -> str:
    name = manifest.get("authorizationProfile", "executor")
    authorization_profile(name)
    return name


def validate_ledger_profile_binding(manifest: dict[str, object], ledger: dict[str, object]) -> None:
    profile_name = manifest_profile_name(manifest)
    policy = authorization_profile(profile_name)
    expected = policy["authorization"]
    if ledger.get("wave") != profile_name or not authorization_matches(ledger.get("authorization"), expected) or not authorization_matches(manifest.get("limits"), expected):
        raise ValueError("ledger authorization profile differs from the pinned preparation profile")


def authorization_matches(actual: object, expected: dict[str, object]) -> bool:
    return isinstance(actual, dict) and stable_json(actual) == stable_json(expected)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def stable_json(value: object) -> bytes:
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n").encode("utf-8")


def atomic_json(path: Path, value: dict[str, object]) -> None:
    ensure_real_directory_chain(path.parent, mode=0o700)
    descriptor, temporary = tempfile.mkstemp(prefix="." + path.name + ".", dir=path.parent)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as handle:
            handle.write(stable_json(value))
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
        directory_fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
    except BaseException:
        try:
            os.unlink(temporary)
        except FileNotFoundError:
            pass
        raise


@contextmanager
def ledger_lock(path: Path):
    lock_path = path.with_name(path.name + ".lock")
    ensure_real_directory_chain(path.parent, mode=0o700)
    descriptor = os.open(lock_path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        fcntl.flock(descriptor, fcntl.LOCK_EX)
        yield
    finally:
        fcntl.flock(descriptor, fcntl.LOCK_UN)
        os.close(descriptor)


def assert_no_symlink_chain(path: Path, *, allow_missing_leaf: bool = False) -> Path:
    absolute = Path(os.path.abspath(path))
    current = Path(absolute.anchor)
    parts = absolute.parts[1:]
    for index, part in enumerate(parts):
        current /= part
        try:
            item = current.lstat()
        except FileNotFoundError:
            if allow_missing_leaf and index == len(parts) - 1:
                return absolute
            raise
        if stat.S_ISLNK(item.st_mode):
            raise ValueError(f"refusing symlink in protected path: {current}")
        if index < len(parts) - 1 and not stat.S_ISDIR(item.st_mode):
            raise ValueError(f"protected path ancestor is not a directory: {current}")
    return absolute


def ensure_real_directory_chain(path: Path, mode: int = 0o700) -> Path:
    absolute = Path(os.path.abspath(path))
    current = Path(absolute.anchor)
    for part in absolute.parts[1:]:
        current /= part
        try:
            item = current.lstat()
        except FileNotFoundError:
            current.mkdir(mode=mode)
            item = current.lstat()
        if stat.S_ISLNK(item.st_mode) or not stat.S_ISDIR(item.st_mode):
            raise ValueError(f"refusing unsafe evidence directory component: {current}")
    return absolute


def read_json(path: Path) -> object:
    return json.loads(path.read_text(encoding="utf-8"), object_pairs_hook=unique_object)


def unique_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON property {key!r}")
        result[key] = value
    return result


def canonical_string(value: str) -> str:
    if any(0xD800 <= ord(char) <= 0xDFFF for char in value):
        raise ValueError("RFC 8785 input contains an unpaired surrogate")
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def canonical_number(value: int | float) -> str:
    if isinstance(value, bool):
        raise TypeError("boolean is not a number here")
    if isinstance(value, int):
        if abs(value) > (2**53 - 1):
            raise ValueError("RFC 8785 plan number is outside the I-JSON safe-integer range")
        return str(value)
    if not math.isfinite(value):
        raise ValueError("RFC 8785 does not permit NaN or infinity")
    if value == 0:
        return "0"

    negative = value < 0
    text = repr(abs(value)).lower()
    if "e" in text:
        mantissa, exponent_text = text.split("e", 1)
        exponent = int(exponent_text)
    else:
        mantissa, exponent = text, 0
    whole, dot, fraction = mantissa.partition(".")
    raw_digits = (whole + fraction) if dot else whole
    decimal_position = len(whole) + exponent
    leading_zeroes = len(raw_digits) - len(raw_digits.lstrip("0"))
    digits = raw_digits.lstrip("0") or "0"
    decimal_position -= leading_zeroes
    while len(digits) > 1 and digits.endswith("0"):
        digits = digits[:-1]

    if 1e-6 <= abs(value) < 1e21:
        if decimal_position <= 0:
            rendered = "0." + "0" * (-decimal_position) + digits
        elif decimal_position >= len(digits):
            rendered = digits + "0" * (decimal_position - len(digits))
        else:
            rendered = digits[:decimal_position] + "." + digits[decimal_position:]
    else:
        exponent = decimal_position - 1
        rendered = digits[0]
        if len(digits) > 1:
            rendered += "." + digits[1:]
        rendered += "e" + ("+" if exponent >= 0 else "") + str(exponent)
    return ("-" if negative else "") + rendered


def canonical_json(value: object) -> bytes:
    if value is None:
        return b"null"
    if value is True:
        return b"true"
    if value is False:
        return b"false"
    if isinstance(value, str):
        return canonical_string(value).encode("utf-8")
    if isinstance(value, (int, float)):
        return canonical_number(value).encode("ascii")
    if isinstance(value, list):
        return b"[" + b",".join(canonical_json(item) for item in value) + b"]"
    if isinstance(value, dict):
        if not all(isinstance(key, str) for key in value):
            raise ValueError("RFC 8785 object keys must be strings")
        ordered = sorted(value, key=lambda key: key.encode("utf-16be"))
        return b"{" + b",".join(
            canonical_string(key).encode("utf-8") + b":" + canonical_json(value[key])
            for key in ordered
        ) + b"}"
    raise ValueError(f"unsupported RFC 8785 value type: {type(value).__name__}")


def snapshot_tree(root: Path) -> dict[str, object]:
    root = assert_no_symlink_chain(root)
    if not stat.S_ISDIR(root.lstat().st_mode):
        raise ValueError("snapshot root is not a real directory")
    entries: list[dict[str, object]] = []

    def visit(directory: Path, relative: str) -> None:
        info = directory.lstat()
        entries.append({"path": relative or ".", "type": "directory", "mode": stat.S_IMODE(info.st_mode)})
        for child in sorted(directory.iterdir(), key=lambda item: item.name):
            child_relative = child.name if not relative else relative + "/" + child.name
            item = child.lstat()
            mode = stat.S_IMODE(item.st_mode)
            if stat.S_ISDIR(item.st_mode):
                visit(child, child_relative)
            elif stat.S_ISREG(item.st_mode):
                data = child.read_bytes()
                entries.append({
                    "path": child_relative,
                    "type": "file",
                    "mode": mode,
                    "size": len(data),
                    "sha256": sha256(data),
                    "base64": base64.b64encode(data).decode("ascii"),
                })
            elif stat.S_ISLNK(item.st_mode):
                target = os.readlink(child)
                entries.append({"path": child_relative, "type": "symlink", "mode": mode, "target": target})
            else:
                raise ValueError(f"fixture snapshot refuses special file {child_relative!r}")

    visit(root, "")
    entries.sort(key=lambda entry: str(entry["path"]).encode("utf-8"))
    return {"schemaVersion": 1, "entries": entries}


def write_snapshot(root: Path, destination: Path) -> dict[str, object]:
    root = assert_no_symlink_chain(root)
    destination = Path(os.path.abspath(destination))
    ensure_real_directory_chain(destination.parent, mode=0o700)
    if destination.exists() or destination.is_symlink():
        raise ValueError(f"refusing to overwrite snapshot evidence: {destination}")
    if destination == root or root in destination.parents:
        raise ValueError("snapshot evidence must be outside the target tree")
    value = snapshot_tree(root)
    destination.write_bytes(stable_json(value))
    return value


def snapshot_index(value: object) -> dict[str, dict[str, object]]:
    if not isinstance(value, dict) or value.get("schemaVersion") != 1 or not isinstance(value.get("entries"), list):
        raise ValueError("invalid snapshot schema")
    index: dict[str, dict[str, object]] = {}
    for item in value["entries"]:
        if not isinstance(item, dict) or not isinstance(item.get("path"), str):
            raise ValueError("invalid snapshot entry")
        path = item["path"]
        if path in index:
            raise ValueError(f"duplicate snapshot path {path!r}")
        index[path] = item
    return index


def compare_snapshots(before: object, after: object, allow_added: set[str] = frozenset()) -> list[str]:
    old = snapshot_index(before)
    new = snapshot_index(after)
    problems: list[str] = []
    for path in sorted(old.keys() | new.keys()):
        if path not in old:
            if path not in allow_added:
                problems.append(f"unexpected added path: {path}")
        elif path not in new:
            problems.append(f"removed path: {path}")
        elif old[path] != new[path]:
            problems.append(f"changed path: {path}")
    return problems


def plan_from(value: object) -> dict[str, object]:
    if isinstance(value, dict) and isinstance(value.get("plan"), dict):
        value = value["plan"]
    if not isinstance(value, dict):
        raise ValueError("expected a plan object or an adoption workflow result containing plan")
    return value


def check_plan_digest(value: object) -> tuple[dict[str, object], str]:
    plan = plan_from(value)
    recorded = plan.get("planDigest")
    if recorded is None:
        raise ValueError("planDigest is null; this plan is ineligible and has no apply authority")
    if not isinstance(recorded, str) or not recorded.startswith("sha256:"):
        raise ValueError("planDigest does not have the expected sha256 form")
    fields = dict(plan)
    del fields["planDigest"]
    calculated = "sha256:" + sha256(canonical_json(fields))
    if calculated != recorded:
        raise ValueError(f"planDigest mismatch: displayed={recorded}, independently calculated={calculated}")
    return plan, calculated


def verify_assets() -> dict[str, object]:
    case = read_json(GREENFIELD / "case.json")
    preserve = read_json(PRESERVATION / "cases.json")
    if not isinstance(case, dict) or not isinstance(preserve, dict):
        raise ValueError("fixture case metadata must be JSON objects")
    if tuple(case["firstRun"]["creates"]) != (
        "engineering/README.md", "engineering/product/overview.md", "docs/README.md", "docs/reference/usage.md"
    ):
        raise ValueError("greenfield fixture's independent four-output set drifted")
    for path, expected_hash in OUTPUT_HASHES.items():
        data = (GREENFIELD / "expected" / path).read_bytes()
        if sha256(data) != expected_hash:
            raise ValueError(f"independent expected output hash mismatch: {path}")
    actual_overlays: dict[str, str] = {}
    for name, (source, _, expected_hash) in OVERLAYS.items():
        data = (FIXTURES / source).read_bytes()
        if sha256(data) != expected_hash:
            raise ValueError(f"independent preservation overlay hash mismatch: {source}")
        actual_overlays[name] = expected_hash
    shared = preserve["sharedBaseline"]["targetFiles"]
    for item in shared:
        data = (FIXTURES / item["source"]).read_bytes()
        if sha256(data) != item["sha256"]:
            raise ValueError(f"independent source fixture hash mismatch: {item['path']}")
    return {
        "expectedOutputs": OUTPUT_HASHES,
        "sourceFiles": {item["path"]: item["sha256"] for item in shared},
        "overlays": actual_overlays,
        "factsSha256": sha256((GREENFIELD / "facts.json").read_bytes()),
        "preservationCaseCount": len(preserve["cases"]),
    }


def verify_baseline() -> None:
    for reference in ("HEAD", "origin/main"):
        completed = subprocess.run(
            ["git", "-C", str(ROOT), "rev-parse", reference],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        if completed.stdout.strip() != BASELINE:
            raise ValueError(f"{reference} is {completed.stdout.strip()}, expected exact S10 baseline {BASELINE}")
    dirty = subprocess.run(
        ["git", "-C", str(ROOT), "status", "--porcelain", "--untracked-files=no"],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    ).stdout
    if dirty:
        raise ValueError("tracked repository files differ from the pinned baseline")


def put_overlay(target: Path, source_relative: str, target_relative: str, expected_hash: str) -> None:
    source = FIXTURES / source_relative
    data = source.read_bytes()
    if sha256(data) != expected_hash:
        raise ValueError(f"overlay changed since fixture declaration: {source_relative}")
    destination = target / target_relative
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_bytes(data)


def prepare(output: Path, profile_name: str = "executor") -> None:
    output = output.expanduser().absolute()
    if output.exists() and any(output.iterdir()):
        raise ValueError(f"refusing to overwrite nonempty preparation directory: {output}")
    policy = authorization_profile(profile_name)
    authorization = dict(policy["authorization"])
    verify_baseline()
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    output.chmod(0o700)
    fixture_hashes = verify_assets()
    targets: dict[str, dict[str, str]] = {"root-cli": {}, "skill": {}}
    snapshot_paths: dict[str, str] = {}

    for channel in ("root-cli", "skill"):
        for case in TARGET_CASES:
            target = output / "targets" / channel / case
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copytree(GREENFIELD / "target", target)
            readme_source, readme_target, readme_hash = OVERLAYS["readme"]
            put_overlay(target, readme_source, readme_target, readme_hash)
            if case == "partial-engineering":
                put_overlay(target, *OVERLAYS[case])
            elif case == "collision":
                put_overlay(target, *OVERLAYS[case])
            elif case == "partial-legacy":
                put_overlay(target, *OVERLAYS[case])
            snapshot_relative = f"evidence/snapshots/before/{channel}-{case}.json"
            write_snapshot(target, output / snapshot_relative)
            targets[channel][case] = str(target.relative_to(output))
            snapshot_paths[f"{channel}/{case}"] = snapshot_relative

    for case in TARGET_CASES:
        root_snapshot = read_json(output / snapshot_paths[f"root-cli/{case}"])
        skill_snapshot = read_json(output / snapshot_paths[f"skill/{case}"])
        if root_snapshot != skill_snapshot:
            raise ValueError(f"CLI/skill {case} targets are not byte-identical at baseline")

    live_override = (ROOT / "assets/flows/adopt.md").read_bytes() + b"\n<!-- S10 disposable live-override marker; created after skill install. -->\n"
    override_seed = output / "evidence/override-seed/adopt.md"
    override_seed.parent.mkdir(parents=True, exist_ok=True)
    override_seed.write_bytes(live_override)
    initial_flow_hash = sha256((ROOT / "assets/flows/adopt.md").read_bytes())
    override_hash = sha256(live_override)
    if override_hash == initial_flow_hash:
        raise ValueError("live override seed must have distinct bytes")

    for channel, cases in (("root-cli", ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"}), ("skill", set(SKILL_CASES))):
        for case in sorted(cases):
            (output / "xdg" / channel / case).mkdir(parents=True, exist_ok=True, mode=0o700)

    ledger = {
        "schemaVersion": 2,
        "baseline": BASELINE,
        "wave": profile_name,
        "authorization": authorization,
        "nativeAuthorization": {
            "state": "blocked",
            "decision": "supplemental-decline profile grants no native sessions" if profile_name == "supplemental-decline" else "N2 pending owner accounting/envelope choice",
        },
        "reservedRootCalls": [],
        "nativeSessions": [],
        "note": (
            "One owner-operated root decline application request maximum; no retry/relaunch/fallback. Native authorization is zero. Await independent offline High review and a fresh root instruction before any reservation."
            if profile_name == "supplemental-decline"
            else "Root calls require one atomically consumed reservation. A failed/interrupted/unknown attempt remains consumed; no retries or revised prompts. Native launch remains blocked pending N2."
        ),
    }
    evidence = output / "evidence"
    evidence.mkdir(parents=True, exist_ok=True)
    (evidence / "ledger.json").write_bytes(stable_json(ledger))
    manifest = {
        "schemaVersion": 2,
        "baseline": BASELINE,
        "preparedRoot": str(output),
        "authorizationProfile": profile_name,
        "fixtureHashes": fixture_hashes,
        "targets": targets,
        "beforeSnapshots": snapshot_paths,
        "flowAsset": {
            "relativeName": "flows/adopt.md",
            "source": "embedded",
            "sha256": initial_flow_hash,
            "overrideSeed": str(override_seed.relative_to(output)),
            "overrideSha256": override_hash,
            "overrideMustBeInstalledOnlyAfterClaudeSkillInstall": True,
            "overrideXdgCases": ["root-cli/override", "skill/override"],
            "allOtherCaseXdgMustRemainEmpty": True,
        },
        "matrix": {
            "rootProviderCases": sorted(policy["rootCases"]),
            "pre-providerVetoCases": ["collision", "partial-legacy"] if profile_name == "executor" else [],
            "no-providerCases": ["noninteractive"] if profile_name == "executor" else [],
            "skillCases": list(SKILL_CASES) if profile_name == "executor" else [],
        },
        "limits": authorization,
        "scope": (
            "separate owner-approved supplemental root decline profile; one application request maximum; no reservation or provider request at preparation; native authorization is zero; independent offline High review and a fresh root instruction are required before reservation"
            if profile_name == "supplemental-decline"
            else "disposable fixture evidence only; native launch blocked pending N2; no pilot, first-release, provider reliability, or Anthropic support claim"
        ),
    }
    (evidence / "prepared.json").write_bytes(stable_json(manifest))
    print(json.dumps({"prepared": str(output), "baseline": BASELINE, "authorizationProfile": profile_name, "rootChatCalls": authorization["rootChatCalls"], "nativeSkillSessions": authorization["nativeSkillSessions"], "targets": 16, "snapshots": 16, "flowAssetSha256": initial_flow_hash, "overrideSeedSha256": override_hash}, sort_keys=True))


def command_prepare(args: argparse.Namespace) -> None:
    prepare(Path(args.out), args.profile)


def command_check_fixtures(_: argparse.Namespace) -> None:
    result = verify_assets()
    expected_results = read_json(ROOT / "internal/cli/testdata/adoption-operations/expected-results.json")
    vectors = expected_results["canonicalizationVectors"] if isinstance(expected_results, dict) else None
    if not isinstance(vectors, list):
        raise ValueError("expected-results lacks digestVectors")
    known_vector = next((item for item in vectors if item.get("purpose", "").startswith("RFC 8785 canonicalizer unit vector")), None)
    if not isinstance(known_vector, dict):
        raise ValueError("independent repository RFC 8785 vector is missing")
    canonical = canonical_json(known_vector["inputJson"])
    if canonical.decode("utf-8") != known_vector["canonicalUtf8"] or sha256(canonical) != known_vector["sha256"]:
        raise ValueError("RFC 8785 repository vector did not match independent canonical bytes/hash")
    numerical_input = json.loads('{"numbers":[333333333.33333329,1E30,4.50,2e-3,0.000000000000000000000000001]}')
    numerical_expected = b'{"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27]}'
    if canonical_json(numerical_input) != numerical_expected:
        raise ValueError("RFC 8785 number-formatting vector mismatch")
    utf16_input = {"\ue000": 1, "\U00010000": 2}
    if canonical_json(utf16_input) != b'{"\xf0\x90\x80\x80":2,"\xee\x80\x80":1}':
        raise ValueError("RFC 8785 UTF-16 property-order vector mismatch")
    print(json.dumps({"fixtureHashes": result, "canonicalizer": "pass", "canonicalizerCases": 3}, sort_keys=True))


def command_snapshot(args: argparse.Namespace) -> None:
    result = write_snapshot(Path(args.root), Path(args.out))
    entries = snapshot_index(result)
    print(json.dumps({"snapshot": str(Path(args.out).resolve()), "entries": len(entries), "sha256": sha256(stable_json(result))}, sort_keys=True))


def command_plan_digest(args: argparse.Namespace) -> None:
    value = read_json(Path(args.input))
    plan, digest = check_plan_digest(value)
    print(json.dumps({"planDigest": digest, "fieldsChecked": len(plan), "canonicalBytesSha256": sha256(canonical_json({k: v for k, v in plan.items() if k != "planDigest"}))}, sort_keys=True))


def load_manifest(output: Path) -> dict[str, object]:
    value = read_json(output / "evidence/prepared.json")
    if not isinstance(value, dict) or value.get("schemaVersion") != 2 or value.get("baseline") != BASELINE:
        raise ValueError("preparation manifest missing or baseline mismatch")
    if value.get("preparedRoot") != str(output):
        raise ValueError("preparation directory differs from its pinned root")
    profile = authorization_profile(manifest_profile_name(value))
    if not authorization_matches(value.get("limits"), profile["authorization"]):
        raise ValueError("preparation limits differ from the selected authorization profile")
    if value.get("matrix", {}).get("rootProviderCases") != sorted(profile["rootCases"]):
        raise ValueError("preparation root cases differ from the selected authorization profile")
    return value


def target_for(output: Path, manifest: dict[str, object], channel: str, case: str) -> Path:
    target_case = "fresh" if case == "rerun" else case
    relative = manifest["targets"][channel].get(target_case)
    if not isinstance(relative, str):
        raise ValueError(f"no prepared {channel}/{target_case} target")
    target = output / relative
    assert_no_symlink_chain(target)
    if not stat.S_ISDIR(target.lstat().st_mode):
        raise ValueError("prepared target is not a real directory")
    return target


def case_xdg(output: Path, channel: str, case: str) -> Path:
    path = output / "xdg" / channel / case
    assert_no_symlink_chain(path)
    if not path.is_dir():
        raise ValueError(f"missing isolated XDG directory for {channel}/{case}")
    return path


def validate_case_xdg(output: Path, manifest: dict[str, object], channel: str, case: str) -> tuple[Path, dict[str, object]]:
    xdg = case_xdg(output, channel, case)
    flow_relative = Path("lds/flows/adopt.md")
    override = xdg / flow_relative
    expected_source = "embedded"
    expected_hash = manifest["flowAsset"]["sha256"]
    if case == "override":
        expected_source = "override"
        expected_hash = manifest["flowAsset"]["overrideSha256"]
        if not override.is_file() or override.is_symlink() or sha256(override.read_bytes()) != expected_hash:
            raise ValueError("override case lacks the exact post-install case-local live-flow override")
        entries = list(xdg.rglob("*"))
        if any(item.is_symlink() or not (item.is_dir() or item.is_file()) for item in entries):
            raise ValueError("case-local override XDG contains a symlink or special file")
        files = {str(item.relative_to(xdg)) for item in entries if item.is_file()}
        directories = {str(item.relative_to(xdg)) for item in entries if item.is_dir()}
        if files != {flow_relative.as_posix()} or directories != {"lds", "lds/flows"}:
            raise ValueError("override XDG contains unexpected files")
    else:
        if any(xdg.rglob("*")):
            raise ValueError(f"non-override XDG for {channel}/{case} must remain empty")
    return xdg, {"source": expected_source, "sha256": expected_hash}


def validate_case_xdg_matrix(output: Path, manifest: dict[str, object]) -> None:
    channels = {
        "root-cli": ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"},
        "skill": set(SKILL_CASES),
    }
    for channel, cases in channels.items():
        for case in cases:
            validate_case_xdg(output, manifest, channel, case)


def safe_environment(api_key: str, xdg: Path, home: Path) -> dict[str, str]:
    allowed = ("PATH", "HOME", "LANG", "LC_ALL", "SSL_CERT_FILE", "SSL_CERT_DIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY")
    environment = {key: os.environ[key] for key in allowed if key in os.environ}
    environment["OPENCODE_API_KEY"] = api_key
    environment["HOME"] = str(home)
    environment["XDG_CONFIG_HOME"] = str(xdg)
    return environment


def verify_plan_shape(plan: object, target: Path, case: str) -> tuple[dict[str, object], str]:
    checked, digest = check_plan_digest(plan)
    required = {
        "schemaVersion", "operation", "targetRoot", "targetIdentity", "planDigest", "factsDigest",
        "outputs", "necessaryParentDirectories", "existingParentDirectories", "preserved", "reportOnly",
        "veto", "findings", "skipped",
    }
    if set(checked) != required:
        raise ValueError(f"plan fields differ from the complete displayed schema: {sorted(set(checked) ^ required)}")
    if checked.get("schemaVersion") != 1 or checked.get("operation") != "docs plan" or checked.get("targetRoot") != str(target):
        raise ValueError("plan schema/operation/target does not bind to the prepared target")
    if not isinstance(checked.get("targetIdentity"), str) or not checked["targetIdentity"]:
        raise ValueError("plan lacks its target identity")
    outputs = checked.get("outputs")
    if not isinstance(outputs, list) or [item.get("path") for item in outputs if isinstance(item, dict)] != list(APPROVED_OUTPUTS):
        raise ValueError("plan does not contain the exact four ordered approved output paths")
    return checked, digest


def validate_case_plan(plan: dict[str, object], case: str) -> None:
    outputs = plan["outputs"]
    classes = [item["classification"] for item in outputs]
    expected = {
        "fresh": (False, ["create"] * 4),
        "rerun": (False, ["identical-untouched"] * 4),
        "partial-engineering": (False, ["create"] * 4),
        "collision": (True, ["create", "different-collision", "create", "create"]),
        "partial-legacy": (True, ["refusal"] * 4),
        "decline": (False, ["create"] * 4),
        "stale": (False, ["create"] * 4),
        "override": (False, ["create"] * 4),
    }
    if case not in expected:
        raise ValueError(f"case {case} has no plan")
    veto, classifications = expected[case]
    if plan.get("veto") is not veto or classes != classifications:
        raise ValueError(f"{case} plan veto/classifications differ from independent fixture expectations")


def command_pin_runtime(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    verify_baseline()
    fixture_hashes = verify_assets()
    if fixture_hashes != manifest["fixtureHashes"]:
        raise ValueError("fixture source/expected bytes changed since preparation")
    binary = assert_no_symlink_chain(Path(args.binary).expanduser())
    if not stat.S_ISREG(binary.lstat().st_mode) or not os.access(binary, os.X_OK):
        raise ValueError("root CLI binary must be an executable regular file")
    pin_path = output / "evidence/runtime-pins.json"
    if pin_path.exists():
        raise ValueError("runtime pins already exist; do not replace the executable pin")
    pins = {
        "schemaVersion": 1,
        "baseline": BASELINE,
        "preparedRoot": str(output),
        "binaryPath": str(binary),
        "binarySha256": sha256(binary.read_bytes()),
        "helperSha256": sha256(Path(__file__).read_bytes()),
        "manifestSha256": sha256((output / "evidence/prepared.json").read_bytes()),
        "fixtureHashes": fixture_hashes,
        "embeddedFlowSha256": manifest["flowAsset"]["sha256"],
        "targetBeforeSnapshotSha256": {
            key: sha256(stable_json(read_json(output / relative)))
            for key, relative in manifest["beforeSnapshots"].items()
        },
        "trackedWorktree": "clean at pinned baseline",
    }
    atomic_json(pin_path, pins)
    print(json.dumps({"runtimePins": str(pin_path.relative_to(output)), "binarySha256": pins["binarySha256"], "trackedWorktree": pins["trackedWorktree"], "liveCalls": 0}, sort_keys=True))


def validate_runtime(output: Path, manifest: dict[str, object], binary: Path) -> dict[str, object]:
    verify_baseline()
    if verify_assets() != manifest["fixtureHashes"]:
        raise ValueError("fixture/source/expected bytes changed since preparation")
    if sha256((ROOT / "assets/flows/adopt.md").read_bytes()) != manifest["flowAsset"]["sha256"]:
        raise ValueError("embedded shared flow changed since preparation")
    pins = read_json(output / "evidence/runtime-pins.json")
    if not isinstance(pins, dict) or pins.get("schemaVersion") != 1 or pins.get("baseline") != BASELINE:
        raise ValueError("missing or invalid executable/runtime pins")
    if pins.get("preparedRoot") != str(output) or pins.get("manifestSha256") != sha256((output / "evidence/prepared.json").read_bytes()):
        raise ValueError("runtime pins do not bind to this preparation manifest")
    if pins.get("binaryPath") != str(binary) or pins.get("binarySha256") != sha256(binary.read_bytes()):
        raise ValueError("root CLI executable differs from its pre-pinned bytes")
    if pins.get("helperSha256") != sha256(Path(__file__).read_bytes()):
        raise ValueError("S10 helper differs from the runtime-pinned bytes")
    if pins.get("fixtureHashes") != manifest["fixtureHashes"] or pins.get("embeddedFlowSha256") != manifest["flowAsset"]["sha256"]:
        raise ValueError("runtime pins do not bind to the independently checked assets")
    if not stat.S_ISREG(binary.lstat().st_mode) or not os.access(binary, os.X_OK):
        raise ValueError("pinned root CLI binary is no longer an executable regular file")
    expected_snapshots = pins.get("targetBeforeSnapshotSha256")
    if not isinstance(expected_snapshots, dict):
        raise ValueError("runtime pins omit complete expected target snapshots")
    for key, relative in manifest["beforeSnapshots"].items():
        expected_hash = sha256(stable_json(read_json(output / relative)))
        if expected_snapshots.get(key) != expected_hash:
            raise ValueError(f"pinned expected before-snapshot changed: {key}")
    return pins


def native_prepared_binding(output: Path, expected_ledger: Path | None = None) -> tuple[Path, dict[str, object], dict[str, object]]:
    prepared = assert_no_symlink_chain(Path(output).expanduser()).resolve(strict=True)
    if not stat.S_ISDIR(prepared.lstat().st_mode):
        raise ValueError("native ledger preparation is not a real directory")
    ledger_path = prepared / "evidence/ledger.json"
    if expected_ledger is not None:
        requested_ledger = assert_no_symlink_chain(Path(expected_ledger).expanduser()).resolve(strict=True)
        if requested_ledger != ledger_path.resolve(strict=True):
            raise ValueError("native ledger transition requires the canonical prepared evidence/ledger.json")

    manifest = load_manifest(prepared)
    profile_name = manifest_profile_name(manifest)
    policy = authorization_profile(profile_name)
    authorization = policy["authorization"]
    if policy["nativeMustRemainBlocked"] or authorization["nativeSkillSessions"] < 1 or authorization["logicalTurnsPerNativeSession"] < 1:
        raise ValueError("prepared authorization profile grants zero native skill sessions")

    pin_path = prepared / "evidence/runtime-pins.json"
    if not pin_path.is_file():
        raise ValueError("missing or invalid executable/runtime pins")
    pins = read_json(pin_path)
    if not isinstance(pins, dict) or not isinstance(pins.get("binaryPath"), str):
        raise ValueError("native ledger transition requires an immutable executable/runtime pin")
    binary = assert_no_symlink_chain(Path(pins["binaryPath"]).expanduser())
    validate_runtime(prepared, manifest, binary)

    ledger = ledger_read(ledger_path)
    validate_ledger_profile_binding(manifest, ledger)
    ledger_validate(ledger)
    return prepared, manifest, ledger


def require_native_settlement(ledger: dict[str, object], case: str) -> dict[str, object]:
    ledger_validate(ledger)
    if ledger.get("nativeAuthorization", {}).get("state") != "authorized":
        raise ValueError("native skill sessions remain blocked pending N2; no settlement allowed")
    matches = [row for row in ledger["nativeSessions"] if row.get("case") == case]
    if len(matches) != 1 or matches[0].get("attemptState") != "consumed" or matches[0].get("outcome") != "unknown":
        raise ValueError("native result must settle exactly one consumed unknown attempt; no retries")
    return matches[0]


def load_flow_result(binary: Path, environment: dict[str, str], expected: dict[str, object]) -> dict[str, object]:
    completed = subprocess.run(
        [str(binary), "plumbing", "adopt", "--json"],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=environment,
        check=False,
    )
    if completed.returncode != 0:
        raise ValueError("read-only live-flow probe failed before root adopt launch")
    value = json.loads(completed.stdout, object_pairs_hook=unique_object)
    if not isinstance(value, dict) or value.get("operation") != "adopt":
        raise ValueError("live-flow probe returned no adoption flow result")
    asset = value.get("asset")
    content = value.get("content")
    if not isinstance(asset, dict) or not isinstance(content, str) or sha256(content.encode("utf-8")) != expected["sha256"]:
        raise ValueError("live flow bytes do not match the case's pinned source/hash")
    if asset.get("source") != expected["source"] or asset.get("sha256") != expected["sha256"]:
        raise ValueError("live-flow source/hash differs from the expected case-local XDG selection")
    return {key: asset.get(key) for key in ("relativeName", "source", "sha256")}


def capture_plan(binary: Path, target: Path, facts: Path, environment: dict[str, str], case: str) -> tuple[dict[str, object], str]:
    completed = subprocess.run(
        [str(binary), "plumbing", "docs", "plan", str(target), "--facts", str(facts), "--json"],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=environment,
        check=False,
    )
    expected_exit = 3 if case in {"collision", "partial-legacy"} else 0
    if completed.returncode != expected_exit:
        raise ValueError(f"read-only prelaunch plan exited {completed.returncode}, expected the fixture-specific {expected_exit}")
    plan, digest = verify_plan_shape(json.loads(completed.stdout, object_pairs_hook=unique_object), target, case)
    validate_case_plan(plan, case)
    expected_finding = {
        "collision": ("refusal.output-collision", "docs/reference/usage.md"),
        "partial-legacy": ("refusal.legacy-docs", "docs/"),
    }.get(case)
    if expected_finding is not None:
        findings = plan.get("findings")
        if not isinstance(findings, list) or len(findings) != 1 or not isinstance(findings[0], dict) or (findings[0].get("code"), findings[0].get("path")) != expected_finding:
            raise ValueError(f"prelaunch veto plan lacks the independently expected case finding {expected_finding}")
    return plan, digest


def validate_tool_diagnostic_bodies(inventory: object, validation: object, exits: object, before: object, case: str) -> None:
    expected_exits = {"inventory": 0, "validate": 1 if case == "collision" else 0}
    if not isinstance(exits, dict) or set(exits) != set(expected_exits) or any(not isinstance(code, int) or isinstance(code, bool) for code in exits.values()) or exits != expected_exits:
        raise ValueError(f"docs inventory/validate exits differ from the pinned {case} fixture: expected {expected_exits}")
    if not isinstance(inventory, dict) or not {"files", "links", "skipped"}.issubset(inventory) or {"schemaVersion", "operation"} & set(inventory):
        raise ValueError("embedded docs inventory body does not match the adopt workflow schema")
    if not isinstance(validation, dict) or not {"findings", "skipped"}.issubset(validation) or {"schemaVersion", "operation"} & set(validation):
        raise ValueError("embedded docs validation body does not match the adopt workflow schema")

    before_entries = snapshot_index(before)
    expected_files = [
        {"path": path, "sha256": entry["sha256"], "size": entry["size"]}
        for path, entry in sorted(before_entries.items())
        if entry.get("type") == "file"
        and path.lower().endswith(".md")
        and (path == "README.md" or path.startswith(("docs/", "engineering/")))
    ]
    def core_items(value: object, fields: tuple[str, ...], name: str, *, optional_line: bool = False) -> list[dict[str, object]]:
        if not isinstance(value, list):
            raise ValueError(f"docs {name} must be an array")
        retained: list[dict[str, object]] = []
        for item in value:
            if not isinstance(item, dict) or any(field not in item for field in fields if not (optional_line and field == "line")):
                raise ValueError(f"docs {name} contains a malformed object")
            retained.append({field: item[field] for field in fields if field in item})
        return retained

    expected_file_core = expected_files
    inventory_file_core = core_items(inventory.get("files"), ("path", "sha256", "size"), "inventory files")
    if any(
        not isinstance(item.get("path"), str)
        or not isinstance(item.get("sha256"), str)
        or not re.fullmatch(r"[0-9a-f]{64}", item["sha256"])
        or not isinstance(item.get("size"), int)
        or isinstance(item.get("size"), bool)
        for item in inventory_file_core
    ) or inventory_file_core != expected_file_core:
        raise ValueError("docs inventory files differ from the complete pinned before snapshot")

    expected_links = []
    expected_findings = []
    if case == "collision":
        expected_links = [{"source": "docs/reference/usage.md", "line": 5, "target": "../howto.md", "kind": "inline"}]
        expected_findings = [{
            "code": "validation.local-target-missing",
            "path": "docs/reference/usage.md",
            "line": 5,
            "target": "../howto.md",
            "message": "local file target is missing: ../howto.md",
        }]
    elif case == "rerun":
        expected_links = [
            {"source": "docs/README.md", "line": 3, "target": "reference/usage.md", "kind": "inline"},
            {"source": "engineering/README.md", "line": 3, "target": "product/overview.md", "kind": "inline"},
        ]
    links = core_items(inventory.get("links"), ("source", "line", "target", "kind"), "inventory links")
    if any(
        not isinstance(item.get("source"), str)
        or not isinstance(item.get("line"), int)
        or isinstance(item.get("line"), bool)
        or item["line"] < 1
        or not isinstance(item.get("target"), str)
        or item.get("kind") not in {"inline", "reference"}
        for item in links
    ) or links != expected_links:
        raise ValueError("docs inventory links are malformed or differ from the exact fixture diagnostics")
    for entries, name in ((inventory.get("skipped"), "inventory"), (validation.get("skipped"), "validation")):
        skipped = core_items(entries, ("source", "line", "code", "reason"), f"{name} skipped", optional_line=True)
        if any(
            not isinstance(item.get("source"), str)
            or ("line" in item and (not isinstance(item["line"], int) or isinstance(item["line"], bool) or item["line"] < 1))
            or not isinstance(item.get("code"), str)
            or not isinstance(item.get("reason"), str)
            for item in skipped
        ) or skipped:
            raise ValueError(f"docs {name} skipped diagnostics are malformed or unexpected for this fixture")
    findings = core_items(validation.get("findings"), ("code", "path", "line", "target", "message"), "validation findings")
    if any(
        not isinstance(item.get("code"), str)
        or not isinstance(item.get("path"), str)
        or not isinstance(item.get("line"), int)
        or isinstance(item.get("line"), bool)
        or item["line"] < 1
        or not isinstance(item.get("target"), str)
        or not isinstance(item.get("message"), str)
        for item in findings
    ) or findings != expected_findings:
        raise ValueError("docs validation findings are malformed or differ from the exact fixture diagnostics")


def standalone_tool_body(value: object, operation: str, fields: set[str]) -> dict[str, object]:
    if not isinstance(value, dict) or value.get("schemaVersion") != 1 or isinstance(value.get("schemaVersion"), bool) or value.get("operation") != operation or not fields.issubset(value):
        raise ValueError(f"standalone {operation} result does not match its operation envelope")
    return {key: item for key, item in value.items() if key not in {"schemaVersion", "operation"}}


def validate_embedded_tool_diagnostics(inventory: object, validation: object, exits: object, before: object, case: str) -> None:
    validate_tool_diagnostic_bodies(inventory, validation, exits, before, case)


def validate_tool_diagnostics(inventory: object, validation: object, exits: object, before: object, case: str) -> None:
    inventory_body = standalone_tool_body(inventory, "docs inventory", {"files", "links", "skipped"})
    validation_body = standalone_tool_body(validation, "docs validate", {"findings", "skipped"})
    validate_tool_diagnostic_bodies(inventory_body, validation_body, exits, before, case)


def capture_tool_diagnostics(binary: Path, target: Path, environment: dict[str, str], before: object, case: str) -> tuple[dict[str, object], dict[str, object], dict[str, int]]:
    results: dict[str, dict[str, object]] = {}
    exits: dict[str, int] = {}
    for name in ("inventory", "validate"):
        completed = subprocess.run(
            [str(binary), "plumbing", "docs", name, str(target), "--json"],
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=environment,
            check=False,
        )
        exits[name] = completed.returncode
        value = json.loads(completed.stdout, object_pairs_hook=unique_object)
        if not isinstance(value, dict):
            raise ValueError(f"actual docs {name} result is not a JSON object")
        results[name] = value
    validate_tool_diagnostics(results["inventory"], results["validate"], exits, before, case)
    if snapshot_tree(target) != before:
        raise ValueError("read-only inventory/validate changed the pinned target before-state")
    return results["inventory"], results["validate"], exits


def actual_cli_diagnostic_regressions(binary: Path, output: Path, manifest: dict[str, object]) -> dict[str, object]:
    cases: dict[str, object] = {}
    retained: dict[str, tuple[dict[str, object], dict[str, object], object]] = {}
    for case in ("fresh", "decline", "collision", "partial-legacy"):
        target = target_for(output, manifest, "root-cli", case)
        before = read_json(output / manifest["beforeSnapshots"][f"root-cli/{case}"])
        environment = safe_environment("OFFLINE_NO_PROVIDER_REQUEST", case_xdg(output, "root-cli", case), output / "home")
        inventory, validation, exits = capture_tool_diagnostics(binary, target, environment, before, case)
        retained[case] = (inventory, validation, before)
        cases[case] = {
            "inventoryExit": exits["inventory"],
            "validateExit": exits["validate"],
            "inventoryLinks": inventory["links"],
            "inventorySkipped": inventory["skipped"],
            "validationFindings": validation["findings"],
            "validationSkipped": validation["skipped"],
        }
        if case in {"collision", "partial-legacy"}:
            plan, _ = capture_plan(binary, target, GREENFIELD / "facts.json", environment, case)
            if not plan.get("veto"):
                raise ValueError(f"actual {case} plan did not preserve the independent all-create veto")
            cases[case]["planExit"] = 3
            cases[case]["planVeto"] = True
            cases[case]["planFinding"] = {"code": plan["findings"][0]["code"], "path": plan["findings"][0]["path"]}
    inventory, validation, before = retained["decline"]
    mutations = (
        ("inventory.links string", {**inventory, "links": ["not-a-link-object"]}, validation),
        ("inventory.skipped string", {**inventory, "skipped": ["not-a-skip-object"]}, validation),
        ("validate.findings string", inventory, {**validation, "findings": ["not-a-finding-object"]}),
        ("validate.skipped string", inventory, {**validation, "skipped": ["not-a-skip-object"]}),
        ("collision finding changed", retained["collision"][0], {**retained["collision"][1], "findings": []}),
    )
    rejected: list[str] = []
    for label, mutated_inventory, mutated_validation in mutations:
        case = "collision" if label == "collision finding changed" else "decline"
        exits = {"inventory": 0, "validate": 1 if case == "collision" else 0}
        try:
            validate_tool_diagnostics(mutated_inventory, mutated_validation, exits, retained[case][2], case)
        except ValueError:
            rejected.append(label)
        else:
            raise ValueError(f"actual tool diagnostic regression accepted mutation: {label}")
    try:
        validate_tool_diagnostics(
            retained["collision"][0], retained["collision"][1],
            {"inventory": 0, "validate": True}, retained["collision"][2], "collision",
        )
    except ValueError:
        rejected.append("boolean exit code")
    else:
        raise ValueError("actual tool diagnostic regression accepted a boolean as process exit 1")
    for case in ("fresh", "decline", "collision", "partial-legacy"):
        inventory_body = standalone_tool_body(retained[case][0], "docs inventory", {"files", "links", "skipped"})
        validation_body = standalone_tool_body(retained[case][1], "docs validate", {"findings", "skipped"})
        validate_embedded_tool_diagnostics(inventory_body, validation_body, {
            "inventory": 0, "validate": 1 if case == "collision" else 0,
        }, retained[case][2], case)
    extension_body = {
        **standalone_tool_body(retained["decline"][0], "docs inventory", {"files", "links", "skipped"}),
        "futureDiagnosticMetadata": {"identity": "preserved-extension"},
    }
    extension_probe = {"schemaVersion": 1, "operation": "docs inventory", **extension_body}
    if standalone_tool_body(extension_probe, "docs inventory", {"files", "links", "skipped"}) != extension_body:
        raise ValueError("diagnostic body comparison dropped a non-wrapper future field")
    altered_extension = {**extension_body, "futureDiagnosticMetadata": {"identity": "changed-extension"}}
    if altered_extension == standalone_tool_body(extension_probe, "docs inventory", {"files", "links", "skipped"}):
        raise ValueError("diagnostic body comparison ignored a changed future field")
    try:
        validate_embedded_tool_diagnostics(
            {**extension_body, "schemaVersion": 1, "operation": "docs inventory"},
            standalone_tool_body(retained["decline"][1], "docs validate", {"findings", "skipped"}),
            {"inventory": 0, "validate": 0}, retained["decline"][2], "decline",
        )
    except ValueError:
        pass
    else:
        raise ValueError("embedded workflow body accepted a standalone schema envelope")
    return {
        "actualBinarySha256": sha256(binary.read_bytes()),
        "providerRequests": 0,
        "nativeSessions": 0,
        "cases": cases,
        "asymmetricMutationsRejected": rejected,
        "embeddedBodyBoundary": "actual root workflow bodies accepted for all four baseline cases; only schemaVersion/operation removed from standalone probes; future body fields retained",
        "scope": "actual baseline read-only plumbing commands; not a human-reviewed S10 root control or provider parity result",
    }


def offline_skill_collection_regression(binary: Path, output: Path) -> dict[str, object]:
    with tempfile.TemporaryDirectory(prefix="lds-bds251-s10-collection-") as temporary:
        clone = Path(temporary) / "prepared"
        shutil.copytree(output, clone)
        manifest_path = clone / "evidence/prepared.json"
        manifest = read_json(manifest_path)
        manifest["preparedRoot"] = str(clone)
        atomic_json(manifest_path, manifest)
        (clone / "evidence/runtime-pins.json").unlink()
        with redirect_stdout(io.StringIO()):
            command_pin_runtime(argparse.Namespace(prepared=str(clone), binary=str(binary)))
        manifest = load_manifest(clone)

        ledger_path = clone / "evidence/ledger.json"
        ledger = ledger_read(ledger_path)
        ledger["nativeAuthorization"] = {"state": "authorized", "decision": "offline synthetic collector regression only"}
        atomic_json(ledger_path, ledger)
        with redirect_stdout(io.StringIO()):
            command_ledger_reserve_skill(argparse.Namespace(ledger=str(ledger_path), case="decline"))
            command_ledger_consume_skill(argparse.Namespace(ledger=str(ledger_path), case="decline"))

        case = "decline"
        target = target_for(clone, manifest, "skill", case)
        before = expected_case_before(clone, manifest, "skill", case)
        environment = safe_environment("OFFLINE_TEST_ONLY", case_xdg(clone, "skill", case), clone / "home")
        flow_completed = subprocess.run(
            [str(binary), "plumbing", "adopt", "--json"],
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=environment,
            check=False,
        )
        if flow_completed.returncode != 0:
            raise ValueError("offline actual shared-flow reader failed in the staged skill-collection regression")
        flow = json.loads(flow_completed.stdout, object_pairs_hook=unique_object)
        if not isinstance(flow, dict) or flow.get("schemaVersion") != 1 or flow.get("operation") != "adopt" or flow.get("status") != "preparatory" or flow.get("availability") != "unavailable-for-execution":
            raise ValueError("offline actual shared-flow reader returned the wrong preparatory schema")
        inventory, validation, exits = capture_tool_diagnostics(binary, target, environment, before, case)
        completed = subprocess.run(
            [str(binary), "plumbing", "docs", "render", str(target), "--facts", str(GREENFIELD / "facts.json"), "--json"],
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=environment,
            check=False,
        )
        if completed.returncode != 0:
            raise ValueError("offline actual docs render failed in the staged skill-collection regression")
        rendered = json.loads(completed.stdout, object_pairs_hook=unique_object)
        plan, digest = capture_plan(binary, target, GREENFIELD / "facts.json", environment, case)

        staging = clone / "evidence/collection-staging/skill" / case
        tool_dir = staging / "tool-results"
        tool_dir.mkdir(parents=True, mode=0o700)
        atomic_json(staging / "flow-result.json", flow)
        atomic_json(staging / "reviewed-plan.json", plan)
        atomic_json(staging / "before-snapshot.json", before)
        for filename, value in (("inventory.json", inventory), ("validate.json", validation), ("render.json", rendered)):
            atomic_json(tool_dir / filename, value)

        skill_root = clone / "home/.claude/skills/lds"
        session = {
            "schemaVersion": 1,
            "channel": "skill",
            "case": case,
            "claudeCodeVersion": "2.1.286",
            "provider": "OpenCode Go Anthropic Messages compatibility gateway",
            "model": "deepseek-v4-flash",
            "sessionExitCode": 0,
            "operationExitCode": expected_operation_exit(case),
            "logicalModelTurns": 1,
            "toolExitCodes": {"adopt": 0, "inventory": exits["inventory"], "validate": exits["validate"], "render": 0, "plan": 0},
            "skillInvocation": "/lds",
            "manualPermissionReview": True,
            "installedSkillSha256": sha256((skill_root / "SKILL.md").read_bytes()),
            "visibleRetryEvents": "none",
            "outcome": "declined",
        }
        atomic_json(staging / "session-evidence.json", session)
        atomic_json(staging / "human-checkpoint.json", {
            "schemaVersion": 1,
            "testOnly": True,
            "synthetic": True,
            "ownerAuthored": True,
            "channel": "skill",
            "case": case,
            "providerConsent": "send",
            "providerConsentPersonallyTyped": True,
            "reviewedPlanDigest": digest,
            "fullPlanPersonallyReviewed": True,
            "decision": "decline",
            "decisionPersonallyTyped": True,
            "typedDecision": "decline",
        })

        with redirect_stdout(io.StringIO()):
            command_ledger_settle_skill(argparse.Namespace(prepared=str(clone), case=case, session_evidence=str(staging / "session-evidence.json")))
        collection_args = argparse.Namespace(
            prepared=str(clone), case=case,
            flow_result=str(staging / "flow-result.json"),
            plan=str(staging / "reviewed-plan.json"),
            before=str(staging / "before-snapshot.json"),
            tool_results=str(tool_dir),
            session_evidence=str(staging / "session-evidence.json"),
            human_checkpoint=str(staging / "human-checkpoint.json"),
            operation_exit_code=expected_operation_exit(case), session_exit_code=0, turns=1,
            retry_events="none", outcome="declined",
        )
        bad_path_args = argparse.Namespace(**vars(collection_args))
        bad_path_args.flow_result = str(clone / "evidence/live/skill/decline/flow-result.json")
        try:
            command_sanitize_skill(bad_path_args)
        except ValueError as error:
            if "outside-target staging" not in str(error):
                raise
        else:
            raise ValueError("offline staged collection accepted a finalized-evidence path as a staging input")
        with redirect_stdout(io.StringIO()):
            command_sanitize_skill(collection_args)
        result_path = clone / "evidence/live/skill/decline/result.json"
        result = read_json(result_path)
        final_dir = result_path.parent
        if result.get("collectionStatus") != "collected-not-verified" or result.get("provider") != "OpenCode Go Anthropic Messages compatibility gateway" or (final_dir / "verified.json").exists():
            raise ValueError("offline staged collection did not remain explicitly unverified")
        if snapshot_tree(target) != before or read_json(clone / result["afterSnapshot"]) != before:
            raise ValueError("offline actual decline collection changed the fixture target")
        try:
            command_sanitize_skill(collection_args)
        except ValueError as error:
            if "already exists" not in str(error) and "cannot be retried" not in str(error):
                raise
        else:
            raise ValueError("offline staged collection regression allowed a result overwrite/retry")
        decline_result = result
        decline_target_unchanged = snapshot_tree(target) == before and read_json(clone / result["afterSnapshot"]) == before
        if not decline_target_unchanged:
            raise ValueError("offline actual decline collection changed the fixture target")

        case = "fresh"
        with redirect_stdout(io.StringIO()):
            command_ledger_reserve_skill(argparse.Namespace(ledger=str(ledger_path), case=case))
            command_ledger_consume_skill(argparse.Namespace(ledger=str(ledger_path), case=case))
        target = target_for(clone, manifest, "skill", case)
        before = expected_case_before(clone, manifest, "skill", case)
        environment = safe_environment("OFFLINE_TEST_ONLY", case_xdg(clone, "skill", case), clone / "home")
        flow_completed = subprocess.run(
            [str(binary), "plumbing", "adopt", "--json"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=environment, check=False,
        )
        if flow_completed.returncode != 0:
            raise ValueError("offline actual shared-flow reader failed in the staged apply regression")
        flow = json.loads(flow_completed.stdout, object_pairs_hook=unique_object)
        inventory, validation, exits = capture_tool_diagnostics(binary, target, environment, before, case)
        rendered_completed = subprocess.run(
            [str(binary), "plumbing", "docs", "render", str(target), "--facts", str(GREENFIELD / "facts.json"), "--json"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=environment, check=False,
        )
        if rendered_completed.returncode != 0:
            raise ValueError("offline actual docs render failed in the staged apply regression")
        rendered = json.loads(rendered_completed.stdout, object_pairs_hook=unique_object)
        plan, digest = capture_plan(binary, target, GREENFIELD / "facts.json", environment, case)
        staging = clone / "evidence/collection-staging/skill" / case
        tool_dir = staging / "tool-results"
        tool_dir.mkdir(parents=True, mode=0o700)
        plan_path = staging / "reviewed-plan.json"
        atomic_json(staging / "flow-result.json", flow)
        atomic_json(plan_path, plan)
        atomic_json(staging / "before-snapshot.json", before)
        for filename, value in (("inventory.json", inventory), ("validate.json", validation), ("render.json", rendered)):
            atomic_json(tool_dir / filename, value)

        apply_completed = subprocess.run(
            [str(binary), "plumbing", "docs", "apply", str(target), "--facts", str(GREENFIELD / "facts.json"), "--plan", str(plan_path), "--confirm", digest, "--json"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=environment, check=False,
        )
        if apply_completed.returncode != 0:
            raise ValueError("offline actual docs apply failed in the staged single-source regression")
        actual_apply = json.loads(apply_completed.stdout, object_pairs_hook=unique_object)
        actual_apply = safe_apply_result(actual_apply)
        if not isinstance(actual_apply, dict) or actual_apply.get("status") != "applied" or actual_apply.get("planDigest") != digest:
            raise ValueError("actual baseline docs apply result was not applied and bound to the captured plan")
        atomic_json(tool_dir / "apply.json", actual_apply)
        after = snapshot_tree(target)
        verify_created_outputs(actual_apply, digest, after)
        additions = {path: "file" for path in APPROVED_OUTPUTS}
        additions.update({path: "directory" for path in expected_added_directories(APPROVED_OUTPUTS, before)})
        if verify_snapshot_delta(before, after, additions):
            raise ValueError("actual offline staged apply changed paths outside the four expected outputs and parents")
        validate_skill_tool_results(
            {filename: read_json(tool_dir / filename) for filename in ("inventory.json", "validate.json", "render.json", "apply.json")},
            target, before, case, digest,
        )

        skill_root = clone / "home/.claude/skills/lds"
        session = {
            "schemaVersion": 1, "testOnly": True, "synthetic": True,
            "channel": "skill", "case": case,
            "claudeCodeVersion": "2.1.286",
            "provider": "OpenCode Go Anthropic Messages compatibility gateway",
            "model": "deepseek-v4-flash",
            "sessionExitCode": 0, "operationExitCode": 0, "logicalModelTurns": 1,
            "toolExitCodes": {"adopt": 0, "inventory": exits["inventory"], "validate": exits["validate"], "render": 0, "plan": 0, "apply": apply_completed.returncode},
            "skillInvocation": "/lds", "manualPermissionReview": True,
            "installedSkillSha256": sha256((skill_root / "SKILL.md").read_bytes()),
            "visibleRetryEvents": "none", "outcome": "applied",
        }
        atomic_json(staging / "session-evidence.json", session)
        atomic_json(staging / "human-checkpoint.json", {
            "schemaVersion": 1, "testOnly": True, "synthetic": True, "ownerAuthored": True,
            "channel": "skill", "case": case, "providerConsent": "send",
            "providerConsentPersonallyTyped": True, "reviewedPlanDigest": digest,
            "fullPlanPersonallyReviewed": True, "decision": "apply",
            "decisionPersonallyTyped": True, "typedDecision": f"apply {digest}",
        })
        with redirect_stdout(io.StringIO()):
            command_ledger_settle_skill(argparse.Namespace(prepared=str(clone), case=case, session_evidence=str(staging / "session-evidence.json")))
        collection_args = argparse.Namespace(
            prepared=str(clone), case=case,
            flow_result=str(staging / "flow-result.json"), plan=str(plan_path),
            before=str(staging / "before-snapshot.json"), tool_results=str(tool_dir),
            session_evidence=str(staging / "session-evidence.json"),
            human_checkpoint=str(staging / "human-checkpoint.json"),
            operation_exit_code=0, session_exit_code=0, turns=1,
            retry_events="none", outcome="applied",
        )

        invalid_apply_cases = ("missing", "malformed", "different-digest", "wrong-output-hash")
        for mutation in invalid_apply_cases:
            bad_output = Path(temporary) / f"prepared-{mutation}"
            shutil.copytree(clone, bad_output)
            bad_manifest_path = bad_output / "evidence/prepared.json"
            bad_manifest = read_json(bad_manifest_path)
            bad_manifest["preparedRoot"] = str(bad_output)
            atomic_json(bad_manifest_path, bad_manifest)
            bad_stage = bad_output / "evidence/collection-staging/skill/fresh"
            bad_apply_path = bad_stage / "tool-results/apply.json"
            if mutation == "missing":
                bad_apply_path.unlink()
            elif mutation == "malformed":
                bad_apply_path.write_text('{"schemaVersion":1,"operation":"docs apply"}\n', encoding="utf-8")
            elif mutation == "wrong-output-hash":
                changed_apply = dict(actual_apply)
                changed_apply["created"] = [dict(entry) for entry in actual_apply["created"]]
                changed_apply["created"][0]["sha256"] = "0" * 64
                atomic_json(bad_apply_path, changed_apply)
            else:
                changed_apply = dict(actual_apply)
                changed_apply["planDigest"] = "sha256:" + "0" * 64
                atomic_json(bad_apply_path, changed_apply)
            bad_args = argparse.Namespace(
                **{
                    **vars(collection_args),
                    "prepared": str(bad_output),
                    "flow_result": str(bad_stage / "flow-result.json"),
                    "plan": str(bad_stage / "reviewed-plan.json"),
                    "before": str(bad_stage / "before-snapshot.json"),
                    "tool_results": str(bad_stage / "tool-results"),
                    "session_evidence": str(bad_stage / "session-evidence.json"),
                    "human_checkpoint": str(bad_stage / "human-checkpoint.json"),
                }
            )
            try:
                command_sanitize_skill(bad_args)
            except (ValueError, json.JSONDecodeError):
                if (bad_output / "evidence/live/skill/fresh/result.json").exists():
                    raise ValueError(f"staged collector laundered the {mutation} actual apply tool artifact")
            else:
                raise ValueError(f"staged collector accepted the {mutation} actual apply tool artifact")

        with redirect_stderr(io.StringIO()):
            try:
                parser().parse_args(["sanitize-skill", "--prepared", str(clone), "--case", "fresh", "--apply-result", str(staging / "apply-result.json")])
            except SystemExit as error:
                if error.code != 2:
                    raise ValueError("removed standalone apply-result option failed for an unexpected parser reason") from None
            else:
                raise ValueError("sanitizer still accepts the removed standalone apply-result duplicate")
        with redirect_stdout(io.StringIO()):
            command_sanitize_skill(collection_args)
        final_dir = clone / "evidence/live/skill/fresh"
        final_result = read_json(final_dir / "result.json")
        final_tool_dir = final_dir / "tool-results"
        if final_result.get("collectionStatus") != "collected-not-verified" or read_json(final_dir / "reviewed-plan.json") != plan or read_json(final_tool_dir / "apply.json") != actual_apply:
            raise ValueError("single-source staged collection did not retain the exact actual reviewed plan/apply artifacts")
        if {item.name for item in final_tool_dir.iterdir()} != {"inventory.json", "validate.json", "render.json", "apply.json"} or (final_dir / "apply-result.json").exists() or (final_tool_dir / "plan.json").exists():
            raise ValueError("single-source staged collection retained a duplicate plan/apply artifact")
        if snapshot_tree(target) != after or read_json(clone / final_result["afterSnapshot"]) != after:
            raise ValueError("single-source staged collection did not retain the complete post-apply snapshot")
        return {
            "case": "decline",
            "applyCase": case,
            "actualBinarySha256": sha256(binary.read_bytes()),
            "providerRequests": 0,
            "nativeSessions": 0,
            "collectionStatus": decline_result["collectionStatus"],
            "applyCollectionStatus": final_result["collectionStatus"],
            "targetUnchanged": decline_target_unchanged,
            "finalizedEvidencePathRejected": True,
            "replacementRefused": True,
            "syntheticSessionAndCheckpoint": True,
            "actualApplyResult": actual_apply,
            "applyToolArtifactNegativesRejected": list(invalid_apply_cases),
            "standaloneApplyResultOptionRemoved": True,
            "singleSourcePlanAndApplyRetained": True,
            "scope": "offline-only staged collector path using the actual baseline LDS flow/inventory/validation/render/plan/apply artifacts; synthetic session/checkpoint are test-only, not native evidence or human consent",
        }


def expected_before_path(output: Path, manifest: dict[str, object], case: str, channel: str = "root-cli") -> Path:
    if case == "rerun":
        verified = output / f"evidence/live/{channel}/fresh/verified.json"
        if not verified.is_file() or read_json(verified).get("outcome") != "PASS":
            raise ValueError("rerun requires a separately verified successful fresh after-snapshot")
        return output / f"evidence/snapshots/after/{channel}-fresh.json"
    return output / manifest["beforeSnapshots"][f"{channel}/{case}"]


def parse_cli_result(raw: bytes) -> dict[str, object] | None:
    if not raw.strip():
        return None
    value = json.loads(raw, object_pairs_hook=unique_object)
    if not isinstance(value, dict):
        raise ValueError("root CLI stdout result is not a JSON object")
    return value


def safe_apply_result(value: object) -> dict[str, object] | None:
    if value is None:
        return None
    if not isinstance(value, dict):
        raise ValueError("root CLI apply result has an invalid shape")
    required = {"schemaVersion", "operation", "planDigest", "status", "created", "identicalUntouched", "unattempted", "failed", "preserved"}
    if not required.issubset(value):
        raise ValueError("root CLI apply result is missing required effect fields")
    if value.get("schemaVersion") != 1 or value.get("operation") != "docs apply" or not isinstance(value.get("planDigest"), str):
        raise ValueError("root CLI apply result is not a digest-bound docs apply result")
    if value.get("status") not in {"applied", "current", "refused", "partial"}:
        raise ValueError("root CLI apply result has an unknown status")
    safe = dict(value)
    for key in ("created", "identicalUntouched", "unattempted", "preserved"):
        entries = value.get(key)
        if not isinstance(entries, list):
            raise ValueError(f"root CLI apply field {key} is missing or not a list")
        if any(not isinstance(item, dict) or not isinstance(item.get("path"), str) for item in entries):
            raise ValueError(f"root CLI apply field {key} contains an invalid path entry")
    failed = value.get("failed")
    if failed is not None and (not isinstance(failed, dict) or not isinstance(failed.get("path"), str) or not isinstance(failed.get("code"), str)):
        raise ValueError("root CLI apply failure must be null or a path/code object")
    return safe


def _root_ledger_change(path: Path, case: str, operation: str, fields: dict[str, object]) -> dict[str, object]:
    with ledger_lock(path):
        ledger = ledger_read(path)
        ledger_validate(ledger)
        matches = [row for row in ledger["reservedRootCalls"] if row.get("case") == case]
        if len(matches) != 1:
            raise ValueError(f"root case {case!r} must have exactly one reservation")
        row = matches[0]
        if operation == "consume":
            if row.get("attemptState") != "reserved" or row.get("attemptOutcome") != "pending":
                raise ValueError("root reservation was already consumed or settled; no retry/relaunch")
            row.update(fields)
            row["attemptState"] = "consumed"
            row["attemptOutcome"] = "unknown"
            row["consumedAtUtc"] = datetime.now(timezone.utc).isoformat()
        elif operation == "settle":
            if row.get("attemptState") != "consumed" or row.get("attemptOutcome") != "unknown":
                raise ValueError("root attempt is not an unsettled consumed reservation")
            row.update(fields)
            row["settledAtUtc"] = datetime.now(timezone.utc).isoformat()
        else:
            raise ValueError("invalid ledger operation")
        ledger_validate(ledger)
        atomic_json(path, ledger)
        return dict(row)


def require_root_reservation(ledger: dict[str, object], case: str) -> dict[str, object]:
    root_calls = ledger_validate(ledger)[0]
    policy = authorization_profile(ledger["wave"])
    reservations = [row for row in ledger["reservedRootCalls"] if row.get("case") == case]
    if case not in policy["rootCases"] or root_calls > policy["authorization"]["rootChatCalls"] or len(reservations) != 1 or reservations[0].get("attemptState") != "reserved" or reservations[0].get("attemptOutcome") != "pending":
        raise ValueError("eligible root launch requires exactly one unused matching reservation within its authorization profile")
    return reservations[0]


class RootRunStopped(Exception):
    pass


def stop_child(child: subprocess.Popen[bytes]) -> None:
    if child.poll() is None:
        try:
            child.terminate()
        except ProcessLookupError:
            pass
        try:
            child.wait(timeout=2)
        except subprocess.TimeoutExpired:
            child.kill()
            child.wait()
    try:
        child.communicate(timeout=1)
    except subprocess.TimeoutExpired:
        child.kill()
        child.communicate()


def command_run_root(args: argparse.Namespace) -> None:
    if args.case not in ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"}:
        raise ValueError("unknown root CLI case")
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    policy = authorization_profile(manifest_profile_name(manifest))
    if args.case not in policy["runCases"]:
        raise ValueError(f"root case {args.case!r} is outside the {manifest_profile_name(manifest)} authorization profile")
    key = os.environ.get("OPENCODE_API_KEY", "")
    if not key:
        raise ValueError("OPENCODE_API_KEY available: false; stop and request root refresh")
    channel = args.channel if args.case == "noninteractive" else "root-cli"
    if args.case != "noninteractive" and args.channel != "root-cli":
        raise ValueError("only the no-inference control may target the skill fixture channel")
    interactive = args.case != "noninteractive"
    if interactive and (not sys.stdin.isatty() or not sys.stderr.isatty()):
        raise ValueError("eligible/control root adopt requires the caller's real terminal on stdin and stderr; no child launched")

    binary = assert_no_symlink_chain(Path(args.binary).expanduser())
    if not stat.S_ISREG(binary.lstat().st_mode) or not os.access(binary, os.X_OK):
        raise ValueError("root CLI binary must be an executable regular file")
    target = target_for(output, manifest, channel, args.case)
    facts = assert_no_symlink_chain(GREENFIELD / "facts.json")
    if sha256(facts.read_bytes()) != manifest["fixtureHashes"]["factsSha256"]:
        raise ValueError("pinned fixture facts changed")
    evidence_dir = output / "evidence/live" / channel / args.case
    attempt_path = evidence_dir / "attempt.json"
    result_path = evidence_dir / "result.json"
    prelaunch_plan_path = evidence_dir / "prelaunch-plan.json"
    reviewed_plan_path = evidence_dir / "reviewed-plan.json"
    before_evidence_path = output / "evidence/snapshots/run-before" / f"{channel}-{args.case}.json"
    after_path = output / "evidence/snapshots/after" / f"{channel}-{args.case}.json"
    paths = (
        result_path,
        attempt_path,
        prelaunch_plan_path,
        reviewed_plan_path,
        before_evidence_path,
        after_path,
    )
    if any(path.exists() for path in paths):
        raise ValueError("root case evidence slot already exists; failed/unknown attempts cannot be retried")
    validate_case_xdg_matrix(output, manifest)
    runtime_pins = validate_runtime(output, manifest, binary)
    xdg, expected_flow = validate_case_xdg(output, manifest, channel, args.case)
    environment = safe_environment(key, xdg, output / "home")
    flow_asset = load_flow_result(binary, environment, expected_flow)

    before_path = expected_before_path(output, manifest, args.case, channel)
    expected_before = read_json(before_path)
    snapshot_index(expected_before)
    current_before = snapshot_tree(target)
    if current_before != expected_before:
        raise ValueError("target bytes/types/modes differ from the pinned expected before-state")

    plan: dict[str, object] | None = None
    digest: str | None = None
    inventory_result: dict[str, object] | None = None
    validation_result: dict[str, object] | None = None
    diagnostic_exits: dict[str, int] | None = None
    if args.case != "noninteractive":
        inventory_result, validation_result, diagnostic_exits = capture_tool_diagnostics(binary, target, environment, current_before, args.case)
        plan, digest = capture_plan(binary, target, facts, environment, args.case)

    ledger_path = output / "evidence/ledger.json"
    if args.case in ROOT_CALL_CASES:
        ledger = ledger_read(ledger_path)
        validate_ledger_profile_binding(manifest, ledger)
        require_root_reservation(ledger, args.case)
    consumed = None

    run_state: dict[str, object] = {"child": None, "settled": False, "interrupted": False, "armed": False, "actualWorkflowDiagnostics": None}
    old_term = signal.getsignal(signal.SIGTERM)
    old_int = signal.getsignal(signal.SIGINT)

    def on_term(_signum: int, _frame: object) -> None:
        run_state["interrupted"] = True
        raise RootRunStopped("SIGTERM")

    def emergency_settle() -> None:
        if run_state["settled"] or not run_state["armed"]:
            return
        child = run_state.get("child")
        if isinstance(child, subprocess.Popen):
            stop_child(child)
        try:
            ensure_real_directory_chain(evidence_dir, mode=0o700)
            if not before_evidence_path.exists():
                atomic_json(before_evidence_path, current_before)
            if not after_path.exists():
                write_snapshot(target, after_path)
            interrupted = bool(run_state["interrupted"])
            outcome = "stopped-unknown" if interrupted else "failed-or-unknown"
            exit_code = 130 if interrupted else None
            atomic_json(attempt_path, {
                "schemaVersion": 1,
                "case": args.case,
                "state": outcome,
                "exitCode": exit_code,
                "applicationRequestOutcome": "unknown",
                "humanInputRelayed": False,
                "rawInputStored": False,
                "rawStreamsStored": False,
                "retryPermitted": False,
                **({"actualWorkflowDiagnostics": run_state["actualWorkflowDiagnostics"]} if run_state["actualWorkflowDiagnostics"] else {}),
            })
            stopped_result: dict[str, object] = {
                "schemaVersion": 2,
                "case": args.case,
                "channel": channel,
                "targetCase": str(target.relative_to(output)),
                "status": outcome,
                "exitCode": exit_code,
                "errorCode": "not-captured-direct-terminal",
                "applicationRequestOutcome": "unknown",
                "logicalProviderCalls": "unknown",
                "providerRequests": "unknown after launch boundary",
                "presentation": "direct caller terminal; harness did not capture stderr",
                "flowAsset": flow_asset,
                "planDigest": digest,
                "beforeSnapshot": str(before_evidence_path.relative_to(output)),
                "beforeSnapshotSha256": sha256(stable_json(current_before)),
                "afterSnapshot": str(after_path.relative_to(output)),
                "rawInputStored": False,
                "rawStreamsStored": False,
            }
            if run_state["actualWorkflowDiagnostics"]:
                stopped_result["actualWorkflowDiagnostics"] = run_state["actualWorkflowDiagnostics"]
            atomic_json(result_path, stopped_result)
            if args.case in ROOT_CALL_CASES:
                ledger = ledger_read(ledger_path)
                rows = [row for row in ledger["reservedRootCalls"] if row.get("case") == args.case]
                if len(rows) == 1 and rows[0].get("attemptState") == "consumed" and rows[0].get("attemptOutcome") == "unknown":
                    _root_ledger_change(ledger_path, args.case, "settle", {
                        "attemptOutcome": outcome,
                        "exitCode": exit_code,
                        "applicationRequestOutcome": "unknown",
                        "retryPermitted": False,
                    })
            run_state["settled"] = True
        except BaseException:
            # Keep the already-consumed reservation as unknown if evidence I/O itself fails.
            pass

    atexit.register(emergency_settle)
    previous_signal_mask = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGINT}) if interactive else None
    try:
        if args.case in ROOT_CALL_CASES:
            consumed = _root_ledger_change(ledger_path, args.case, "consume", {
                "logicalChatRequests": 1,
                "baseline": BASELINE,
                "binarySha256": runtime_pins["binarySha256"],
                "flowAsset": flow_asset,
                "beforeSnapshotSha256": sha256(stable_json(current_before)),
                "planDigest": digest,
            })
        run_state["armed"] = True
        if interactive:
            signal.signal(signal.SIGTERM, on_term)
            signal.signal(signal.SIGINT, on_term)
    finally:
        if previous_signal_mask is not None:
            signal.pthread_sigmask(signal.SIG_SETMASK, previous_signal_mask)
    evidence_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
    atomic_json(attempt_path, {
        "schemaVersion": 1,
        "case": args.case,
        "channel": channel,
        "state": "consumed-before-launch" if consumed else "no-inference-control",
        "humanInputRelayed": False,
        "rawInputStored": False,
        "retryPermitted": False,
        "rawStreamsPersisted": False,
    })
    if plan is not None:
        atomic_json(prelaunch_plan_path, plan)
    atomic_json(before_evidence_path, current_before)

    # Revalidate immediately at the launch boundary, after reservation consumption.
    validate_runtime(output, manifest, binary)
    if snapshot_tree(target) != current_before:
        raise ValueError("target changed between preflight and child launch; reservation remains consumed")
    current_flow = load_flow_result(binary, environment, expected_flow)
    if current_flow != flow_asset:
        raise ValueError("live-flow selection changed immediately before child launch")

    command = [str(binary), "adopt", str(target), "--facts", str(facts), "--json"]
    stdout = b""
    return_code: int | None = None
    stopped = False
    child: subprocess.Popen[bytes] | None = None
    try:
        if interactive:
            child = subprocess.Popen(command, stdin=sys.stdin, stdout=subprocess.PIPE, stderr=sys.stderr, env=environment)
            run_state["child"] = child
            stdout, _ = child.communicate()
            return_code = child.returncode
        else:
            completed = subprocess.run(command, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=environment, check=False)
            stdout = completed.stdout
            return_code = completed.returncode
            if b"refusal.interactive-required" not in completed.stderr:
                raise ValueError("noninteractive control did not return the exact interactive-required refusal")
    except (KeyboardInterrupt, RootRunStopped):
        stopped = True
        run_state["interrupted"] = True
        if child is not None:
            stop_child(child)
        run_state["child"] = None
        stdout = b""
        return_code = 130
    except BaseException:
        if child is not None:
            stop_child(child)
        run_state["child"] = None
        raise
    run_state["child"] = None
    if stopped:
        emergency_settle()
        atexit.unregister(emergency_settle)
        if interactive:
            signal.signal(signal.SIGTERM, old_term)
            signal.signal(signal.SIGINT, old_int)
        raise SystemExit(130)

    try:
        result = parse_cli_result(stdout)
    except (ValueError, json.JSONDecodeError):
        run_state["actualWorkflowDiagnostics"] = "malformed-result-json"
        raise ValueError("actual adopt workflow result was malformed; retain failed/unknown attempt and stop") from None
    if args.case != "noninteractive":
        if result is None:
            run_state["actualWorkflowDiagnostics"] = "missing-actual-result"
            raise ValueError("actual adopt workflow emitted no complete result; retain failed/unknown attempt and stop")
        if result.get("schemaVersion") != 1 or result.get("operation") != "adopt" or result.get("targetRoot") != str(target):
            run_state["actualWorkflowDiagnostics"] = "invalid-actual-result-schema"
            raise ValueError("actual adopt workflow result schema/target is invalid; retain failed/unknown attempt and stop")
        try:
            actual_inventory = result.get("inventory")
            actual_validation = result.get("validation")
            validate_embedded_tool_diagnostics(actual_inventory, actual_validation, diagnostic_exits, current_before, args.case)
            if actual_inventory != standalone_tool_body(inventory_result, "docs inventory", {"files", "links", "skipped"}) or actual_validation != standalone_tool_body(validation_result, "docs validate", {"findings", "skipped"}):
                raise ValueError("actual workflow diagnostics differ from the independent prelaunch probes")
        except ValueError:
            run_state["actualWorkflowDiagnostics"] = "actual-result-diagnostics-rejected"
            raise ValueError("actual adopt workflow inventory/validation is missing, malformed, or differs from its independent prelaunch observations; retain failed/unknown attempt and stop") from None
        run_state["actualWorkflowDiagnostics"] = "matched-independent-prelaunch-probes"
    status = result.get("status") if result else None
    raw_plan = result.get("plan") if result else None
    reviewed_plan: dict[str, object] | None = None
    observed_digest: str | None = None
    if args.case != "noninteractive":
        if raw_plan is None:
            run_state["actualWorkflowDiagnostics"] = "missing-actual-plan"
            raise ValueError("actual adopt workflow result omitted its complete plan; retain failed/unknown attempt and stop")
        try:
            reviewed_plan, observed_digest = verify_plan_shape(raw_plan, target, args.case)
        except ValueError:
            run_state["actualWorkflowDiagnostics"] = "invalid-actual-plan"
            raise ValueError("actual adopt workflow plan is invalid; retain failed/unknown attempt and stop") from None
        if plan is None or reviewed_plan != plan or observed_digest != digest:
            run_state["actualWorkflowDiagnostics"] = "actual-plan-differs-from-prelaunch"
            raise ValueError("actual adopt workflow plan differs from the independent prelaunch plan; retain failed/unknown attempt and stop")
        validate_case_plan(reviewed_plan, args.case)
        atomic_json(reviewed_plan_path, reviewed_plan)
    if digest is not None and observed_digest is not None and digest != observed_digest:
        raise ValueError("root result plan differs from prelaunch fixture plan; retain attempt and stop")

    if args.case == "noninteractive":
        request_outcome: object = 0
        human_review = "not-applicable-no-plan"
    elif reviewed_plan and reviewed_plan["veto"]:
        request_outcome = 0
        human_review = "manual plan-read checkpoint required; no apply decision is eligible"
    elif result and result.get("advisory"):
        request_outcome = 1
        human_review = "owner-authored send and full-plan checkpoint required"
    else:
        request_outcome = "unknown"
        human_review = "owner checkpoint required; provider attempt outcome cannot be inferred"

    safe_result: dict[str, object] = {
        "schemaVersion": 2,
        "case": args.case,
        "channel": channel,
        "targetCase": str(target.relative_to(output)),
        "exitCode": return_code,
        "operationExitCode": return_code,
        "sessionExitCode": None,
        "errorCode": "refusal.interactive-required" if args.case == "noninteractive" else "not-captured-direct-terminal",
        "status": "stopped-unknown" if stopped else status,
        "outcome": "not-started" if channel == "skill" and args.case == "noninteractive" else None,
        "provider": "OpenCode Go",
        "model": "deepseek-v4-flash",
        "endpoint": "https://opencode.ai/zen/go/v1/chat/completions",
        "flowAsset": flow_asset,
        "planDigest": observed_digest,
        "logicalProviderCalls": request_outcome,
        "applicationRequestOutcome": request_outcome,
        "presentation": "direct caller terminal; harness did not capture stderr" if interactive else "not-applicable-no-plan",
        "humanReviewCheckpoint": human_review,
        "rawInputStored": False,
        "rawStreamsStored": False,
        "stdoutBytes": len(stdout),
        "beforeSnapshot": str(before_evidence_path.relative_to(output)),
        "beforeSnapshotSha256": sha256(stable_json(current_before)),
        "afterSnapshot": str(after_path.relative_to(output)),
        "runtimePinsSha256": sha256((output / "evidence/runtime-pins.json").read_bytes()),
    }
    if result:
        if result.get("schemaVersion") != 1 or result.get("operation") != "adopt" or result.get("targetRoot") != str(target):
            raise ValueError("root CLI result schema/operation/target does not match launched case")
        flow_result_asset = result.get("flowAsset")
        if flow_result_asset != flow_asset:
            raise ValueError("root result did not use the pinned live-flow source/hash")
        safe_result["apply"] = safe_apply_result(result.get("apply"))
        safe_result["resultOperation"] = result.get("operation")
        if args.case != "noninteractive":
            safe_result["inventory"] = actual_inventory
            safe_result["validation"] = actual_validation
            safe_result["diagnosticExits"] = diagnostic_exits
            safe_result["actualWorkflowDiagnostics"] = "matched-independent-prelaunch-probes"
    elif args.case in {"collision", "partial-legacy"}:
        raise ValueError("pre-provider veto did not produce a complete root result")
    elif args.case == "noninteractive":
        if stdout:
            raise ValueError("noninteractive refusal unexpectedly produced a plan/result")
    after = snapshot_tree(target)
    write_snapshot(target, after_path)
    atomic_json(result_path, safe_result)
    atomic_json(attempt_path, {
        "schemaVersion": 1,
        "case": args.case,
        "channel": channel,
        "state": "stopped-unknown" if stopped else "settled",
        "exitCode": return_code,
        "applicationRequestOutcome": request_outcome,
        "humanInputRelayed": False,
        "rawInputStored": False,
        "rawStreamsStored": False,
        "retryPermitted": False,
    })
    if consumed:
        _root_ledger_change(output / "evidence/ledger.json", args.case, "settle", {
            "attemptOutcome": "stopped-unknown" if stopped else "completed",
            "exitCode": return_code,
            "applicationRequestOutcome": request_outcome,
            "planDigest": observed_digest,
            "retryPermitted": False,
        })
    run_state["settled"] = True
    atexit.unregister(emergency_settle)
    if interactive:
        signal.signal(signal.SIGTERM, old_term)
        signal.signal(signal.SIGINT, old_int)
    print(json.dumps({
        "case": args.case, "channel": channel, "exitCode": return_code, "status": status,
        "result": str(result_path.relative_to(output)),
        "reviewedPlan": str(reviewed_plan_path.relative_to(output)) if reviewed_plan else None,
        "afterSnapshot": str(after_path.relative_to(output)),
        "applicationRequestOutcome": request_outcome,
        "humanCheckpointRequired": args.case in ROOT_CALL_CASES or bool(reviewed_plan),
        "rawFiles": 0,
    }, sort_keys=True))
    if stopped:
        raise SystemExit(130)


def open_target_root(target: Path) -> int:
    absolute = Path(os.path.abspath(target))
    descriptor = os.open(absolute.anchor, os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in absolute.parts[1:]:
            next_descriptor = os.open(
                part,
                os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW,
                dir_fd=descriptor,
            )
            os.close(descriptor)
            descriptor = next_descriptor
        return descriptor
    except BaseException:
        os.close(descriptor)
        raise


def mkdirat_nofollow(parent_fd: int, name: str) -> int:
    try:
        os.mkdir(name, 0o755, dir_fd=parent_fd)
    except FileExistsError:
        pass
    return os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent_fd)


class StaleControllerRestorationError(OSError):
    pass


def insert_collision_no_follow(target: Path, expected_before: object) -> str:
    if snapshot_tree(target) != expected_before:
        raise ValueError("stale target no longer matches its pinned before-state")
    data = (PRESERVATION / "output-collision/docs/reference/usage.md").read_bytes()
    if sha256(data) != OVERLAYS["collision"][2]:
        raise ValueError("stale collision fixture hash mismatch")
    root_fd = open_target_root(target)
    docs_fd = -1
    reference_fd = -1
    root_mode = stat.S_IMODE(os.fstat(root_fd).st_mode)
    docs_mode: int | None = None
    root_sealed = False
    docs_sealed = False
    try:
        docs_fd = mkdirat_nofollow(root_fd, "docs")
        os.fchmod(root_fd, root_mode & ~0o222)
        root_sealed = True
        root_docs = os.stat("docs", dir_fd=root_fd, follow_symlinks=False)
        if not stat.S_ISDIR(root_docs.st_mode) or (root_docs.st_dev, root_docs.st_ino) != (os.fstat(docs_fd).st_dev, os.fstat(docs_fd).st_ino):
            raise ValueError("stale controller could not establish the pinned target/docs directory relationship")

        reference_fd = mkdirat_nofollow(docs_fd, "reference")
        docs_mode = stat.S_IMODE(os.fstat(docs_fd).st_mode)
        os.fchmod(docs_fd, docs_mode & ~0o222)
        docs_sealed = True
        docs_reference = os.stat("reference", dir_fd=docs_fd, follow_symlinks=False)
        if not stat.S_ISDIR(docs_reference.st_mode) or (docs_reference.st_dev, docs_reference.st_ino) != (os.fstat(reference_fd).st_dev, os.fstat(reference_fd).st_ino):
            raise ValueError("stale controller could not establish the pinned docs/reference directory relationship")

        descriptor = os.open("usage.md", os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o644, dir_fd=reference_fd)
        with os.fdopen(descriptor, "wb") as handle:
            handle.write(data)
            handle.flush()
            os.fsync(handle.fileno())
    finally:
        restoration_errors: list[BaseException] = []
        if docs_sealed and docs_fd >= 0 and docs_mode is not None:
            try:
                os.fchmod(docs_fd, docs_mode)
                if stat.S_IMODE(os.fstat(docs_fd).st_mode) != docs_mode:
                    raise OSError("docs directory mode differs after restoration")
            except BaseException as error:
                restoration_errors.append(error)
        if root_sealed:
            try:
                os.fchmod(root_fd, root_mode)
                if stat.S_IMODE(os.fstat(root_fd).st_mode) != root_mode:
                    raise OSError("target root mode differs after restoration")
            except BaseException as error:
                restoration_errors.append(error)
        if reference_fd >= 0:
            os.close(reference_fd)
        if docs_fd >= 0:
            os.close(docs_fd)
        os.close(root_fd)
        if restoration_errors:
            raise StaleControllerRestorationError("stale controller could not establish original fixture directory modes") from restoration_errors[0]
    return sha256(data)


def command_stale_insert(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    if "stale" not in authorization_profile(manifest_profile_name(manifest))["rootCases"]:
        raise ValueError("stale controller is outside this preparation's authorized root cases")
    if args.channel not in {"root-cli", "skill"} or not args.plan:
        raise ValueError("stale insertion requires a channel-scoped complete prelaunch plan artifact")
    target = target_for(output, manifest, args.channel, "stale")
    supplied_target = Path(os.path.abspath(Path(args.target).expanduser()))
    if supplied_target != target:
        raise ValueError("stale controller only accepts the named disposable stale target")
    expected_before = read_json(output / manifest["beforeSnapshots"][f"{args.channel}/stale"])
    assert_no_symlink_chain(Path(args.plan).expanduser())
    plan, digest = verify_plan_shape(read_json(Path(args.plan)), target, "stale")
    validate_case_plan(plan, "stale")
    if plan["veto"] or len([item for item in plan["outputs"] if item.get("classification") == "create"]) != 4:
        raise ValueError("stale controller requires a complete eligible four-create plan")
    event_path = output / "evidence/events" / f"stale-controller-{args.channel}.json"
    after_path = output / "evidence/snapshots" / f"stale-after-controller-{args.channel}.json"
    if event_path.exists() or after_path.exists():
        raise ValueError("channel stale insertion already exists; no repeated controller action")
    collision_hash = OVERLAYS["collision"][2]
    try:
        inserted_hash = insert_collision_no_follow(target, expected_before)
        after = snapshot_tree(target)
        collision = snapshot_index(after).get("docs/reference/usage.md")
        additions = {"docs": "directory", "docs/reference": "directory", "docs/reference/usage.md": "file"}
        if not collision or collision.get("type") != "file" or collision.get("sha256") != collision_hash or verify_snapshot_delta(expected_before, after, additions):
            raise ValueError("stale controller did not create only the pinned collision inside the named target")
        write_snapshot(target, after_path)
        atomic_json(event_path, {
            "schemaVersion": 2,
            "channel": args.channel,
            "event": "pinned differing-output collision inserted into disposable stale target after operator-triggered review checkpoint",
            "controllerStatus": "inserted",
            "approvalAllowed": True,
            "reviewedPlanDigest": digest,
            "planArtifactSha256": sha256(stable_json(plan)),
            "beforeSnapshot": str((output / manifest["beforeSnapshots"][f"{args.channel}/stale"]).relative_to(output)),
            "afterSnapshot": str(after_path.relative_to(output)),
            "insertedPath": "docs/reference/usage.md",
            "insertedSha256": inserted_hash,
            "controllerObservedTerminalPresentation": False,
            "controllerGeneratedOrForwardedApproval": False,
            "timestampUtc": datetime.now(timezone.utc).isoformat(),
        })
    except BaseException as error:
        category = "permission-restoration-failed" if isinstance(error, StaleControllerRestorationError) else "effective-permissions-not-established" if isinstance(error, PermissionError) else "controller-write-or-state-failed"
        try:
            after = snapshot_tree(target)
            if not after_path.exists():
                write_snapshot(target, after_path)
            atomic_json(event_path, {
                "schemaVersion": 2,
                "channel": args.channel,
                "event": "stale controller failed; STOP WITHOUT APPROVAL",
                "controllerStatus": "failed",
                "approvalAllowed": False,
                "failureCategory": category,
                "reviewedPlanDigest": digest,
                "planArtifactSha256": sha256(stable_json(plan)),
                "beforeSnapshot": str((output / manifest["beforeSnapshots"][f"{args.channel}/stale"]).relative_to(output)),
                "afterSnapshot": str(after_path.relative_to(output)),
                "controllerGeneratedOrForwardedApproval": False,
                "timestampUtc": datetime.now(timezone.utc).isoformat(),
            })
        except BaseException:
            raise ValueError("stale controller failed and could not retain its whole after snapshot; STOP WITHOUT APPROVAL") from None
        raise ValueError(f"stale controller failed ({category}); failure and whole after snapshot retained; STOP WITHOUT APPROVAL") from None
    print(json.dumps({"channel": args.channel, "insertedPath": "docs/reference/usage.md", "sha256": inserted_hash, "snapshotEntries": len(snapshot_index(after)), "planDigest": digest, "next": "return to the still-open original terminal/chat; only the owner may decide whether to type the old displayed digest"}, sort_keys=True))


def expected_added_directories(output_paths: tuple[str, ...], before: object) -> set[str]:
    old = snapshot_index(before)
    expected: set[str] = set()
    for output_path in output_paths:
        parent = Path(output_path).parent
        while parent != Path("."):
            value = parent.as_posix()
            if value not in old:
                expected.add(value)
            parent = parent.parent
    return expected


def verify_snapshot_delta(before: object, after: object, expected_added: dict[str, str]) -> list[str]:
    old = snapshot_index(before)
    new = snapshot_index(after)
    problems: list[str] = []
    for path in sorted(old):
        if path not in new:
            problems.append(f"removed path: {path}")
        elif old[path] != new[path]:
            problems.append(f"changed path: {path}")
    actual_added = set(new) - set(old)
    if actual_added != set(expected_added):
        problems.append(f"added paths {sorted(actual_added)!r} differ from exact expected set {sorted(expected_added)!r}")
    for path, expected_type in expected_added.items():
        if path in new and new[path].get("type") != expected_type:
            problems.append(f"added {path} has type {new[path].get('type')!r}, expected {expected_type!r}")
    return problems


def read_human_checkpoint(path: Path, channel: str, case: str, digest: str | None) -> dict[str, object]:
    value = read_json(path)
    if not isinstance(value, dict) or value.get("schemaVersion") != 1:
        raise ValueError(f"{channel}/{case} owner checkpoint is missing or malformed")
    if value.get("ownerAuthored") is not True or value.get("channel") != channel or value.get("case") != case:
        raise ValueError(f"{channel}/{case} checkpoint is not the owner-authored checkpoint for this case")
    if value.get("reviewedPlanDigest") != digest or value.get("fullPlanPersonallyReviewed") is not True:
        raise ValueError(f"{channel}/{case} checkpoint does not bind personal full-plan review to the result digest")
    return value


def expected_decision(case: str) -> tuple[str, str, str]:
    if case in {"collision", "partial-legacy"}:
        return "not-requested", "veto-no-confirmation", "not-applicable"
    if case == "decline":
        return "send", "decline", "decline"
    return "send", "apply", "apply-digest"


def check_human_decision(checkpoint: dict[str, object], case: str, digest: str) -> None:
    provider_consent, decision, decision_kind = expected_decision(case)
    if checkpoint.get("providerConsent") != provider_consent:
        raise ValueError("owner checkpoint does not record the required provider-transfer decision")
    if provider_consent == "send" and checkpoint.get("providerConsentPersonallyTyped") is not True:
        raise ValueError("owner checkpoint does not attest a personally typed exact send consent")
    personally_typed = decision != "veto-no-confirmation"
    if checkpoint.get("decision") != decision or checkpoint.get("decisionPersonallyTyped") is not personally_typed:
        raise ValueError("owner checkpoint does not record the personally typed expected decision")
    if decision_kind == "apply-digest" and checkpoint.get("typedDecision") != f"apply {digest}":
        raise ValueError("owner checkpoint exact apply line does not match the reviewed digest")
    if decision_kind == "decline" and checkpoint.get("typedDecision") != "decline":
        raise ValueError("decline checkpoint must retain only the categorical value 'decline'")
    if decision_kind == "not-applicable" and checkpoint.get("typedDecision") != "not-applicable":
        raise ValueError("veto checkpoint must not claim an apply decision")


def apply_paths(apply: object, key: str) -> list[str]:
    if not isinstance(apply, dict) or not isinstance(apply.get(key), list):
        raise ValueError(f"apply result lacks the required {key} array")
    paths = [entry.get("path") for entry in apply[key] if isinstance(entry, dict)]
    if len(paths) != len(apply[key]) or not all(isinstance(path, str) for path in paths):
        raise ValueError(f"apply result {key} contains malformed paths")
    return paths


def expected_operation_exit(case: str) -> int:
    return 3 if case in {"collision", "partial-legacy", "decline", "stale"} else 0


def validate_skill_tool_exits(case: str, value: object) -> None:
    if not isinstance(value, dict) or any(not isinstance(code, int) or isinstance(code, bool) for code in value.values()):
        raise ValueError("session evidence lacks actual integer skill-tool process exits")
    expected = {
        "adopt": 0,
        "inventory": 0,
        "validate": 1 if case == "collision" else 0,
        "render": 0,
        "plan": 3 if case in {"collision", "partial-legacy"} else 0,
    }
    if case in {"fresh", "rerun", "partial-engineering", "stale", "override"}:
        expected["apply"] = 3 if case == "stale" else 0
    if value != expected:
        raise ValueError(f"actual skill-tool exits differ from the exact {case} operation set: expected {expected}, received {value}")


def validate_skill_tool_results(values: dict[str, object], target: Path, before: object, case: str, digest: str) -> dict[str, object] | None:
    required = {"inventory.json", "validate.json", "render.json"}
    apply_cases = {"fresh", "rerun", "partial-engineering", "stale", "override"}
    if case in apply_cases:
        required.add("apply.json")
    if set(values) != required:
        raise ValueError(f"actual skill tool-result set is incomplete or has extra artifacts: expected {sorted(required)}, received {sorted(values)}")

    operations = {
        "inventory.json": ("docs inventory", {"files", "links", "skipped"}),
        "validate.json": ("docs validate", {"findings", "skipped"}),
        "render.json": ("docs render", {"outputs"}),
    }
    for filename, (operation, fields) in operations.items():
        value = values.get(filename)
        if not isinstance(value, dict) or value.get("schemaVersion") != 1 or value.get("operation") != operation or not fields.issubset(value):
            raise ValueError(f"actual skill tool result {filename} is missing its operation schema")
        if "targetRoot" in value and value.get("targetRoot") != str(target):
            raise ValueError(f"actual skill tool result {filename} names a different target")

    diagnostic_exits = {"inventory": 0, "validate": 1 if case == "collision" else 0}
    validate_tool_diagnostics(values["inventory.json"], values["validate.json"], diagnostic_exits, before, case)

    render = values["render.json"]
    if not isinstance(render, dict) or not isinstance(render.get("outputs"), list) or len(render["outputs"]) != len(APPROVED_OUTPUTS):
        raise ValueError("actual docs render result does not contain all four outputs")
    for rendered, path in zip(render["outputs"], APPROVED_OUTPUTS, strict=True):
        expected_bytes = (GREENFIELD / "expected" / path).read_bytes()
        if not isinstance(rendered, dict) or rendered.get("path") != path or rendered.get("content") != expected_bytes.decode("utf-8") or rendered.get("sha256") != OUTPUT_HASHES[path] or sha256(rendered["content"].encode("utf-8")) != OUTPUT_HASHES[path]:
            raise ValueError(f"actual docs render result differs from independent fixture bytes: {path}")
        asset = rendered.get("asset")
        if not isinstance(asset, dict) or not all(isinstance(asset.get(key), str) and asset[key] for key in ("relativeName", "source", "sha256")):
            raise ValueError(f"actual docs render result lacks asset provenance: {path}")

    apply = None
    if "apply.json" in values:
        apply = safe_apply_result(values["apply.json"])
        if not isinstance(apply, dict) or apply.get("planDigest") != digest:
            raise ValueError("actual docs apply result is absent or not bound to the reviewed plan")
        expected_status = "refused" if case == "stale" else "current" if case == "rerun" else "applied"
        if apply.get("status") != expected_status:
            raise ValueError("actual docs apply status differs from the independently expected case outcome")
        expected_created = list(APPROVED_OUTPUTS) if case in {"fresh", "partial-engineering", "override"} else []
        expected_created_entries = [{"path": path, "sha256": OUTPUT_HASHES[path]} for path in expected_created]
        if apply.get("created") != expected_created_entries:
            raise ValueError("actual docs apply created paths/hashes differ from the independent fixture expectation")
        if case == "rerun" and sorted(apply_paths(apply, "identicalUntouched")) != sorted(APPROVED_OUTPUTS):
            raise ValueError("actual docs rerun did not preserve all four identical outputs")
    return apply


def expected_skill_outcome(case: str) -> str:
    if case in {"collision", "partial-legacy", "stale"}:
        return "refused"
    if case == "decline":
        return "declined"
    if case == "rerun":
        return "current"
    return "applied"


def validate_skill_session(case: str, session: object, operation_exit: int, session_exit: int, turns: int, retry_events: object, outcome: str) -> dict[str, object]:
    required = (
        "claudeCodeVersion", "provider", "model", "sessionExitCode", "operationExitCode",
        "logicalModelTurns", "toolExitCodes", "skillInvocation", "manualPermissionReview",
        "installedSkillSha256", "visibleRetryEvents", "outcome",
    )
    if not isinstance(session, dict) or session.get("schemaVersion") != 1 or session.get("channel") != "skill" or session.get("case") != case or any(key not in session for key in required):
        raise ValueError("owner session evidence is missing its matching case or required runtime/exit/permission fields")
    validate_retry_events(session.get("visibleRetryEvents"))
    if session.get("skillInvocation") != "/lds" or session.get("manualPermissionReview") is not True:
        raise ValueError("session does not attest actual installed /lds invocation and manually reviewed permissions")
    if session.get("provider") != "OpenCode Go Anthropic Messages compatibility gateway" or session.get("model") != "deepseek-v4-flash" or session.get("claudeCodeVersion") != "2.1.286":
        raise ValueError("skill session differs from the pinned Claude Code/provider/model route")
    observations = (session.get("operationExitCode"), session.get("sessionExitCode"), session.get("logicalModelTurns"))
    if any(not isinstance(value, int) or isinstance(value, bool) for value in observations):
        raise ValueError("session exit/turn counters must be integer observations")
    if observations != (operation_exit, session_exit, turns) or retry_events != session.get("visibleRetryEvents") or outcome != session.get("outcome"):
        raise ValueError("declared session counts/outcome differ from the retained actual session evidence")
    if operation_exit != expected_operation_exit(case) or outcome != expected_skill_outcome(case):
        raise ValueError("native operation outcome differs from this case's independently expected result")
    if session_exit != 0 or turns < 1 or turns > 8:
        raise ValueError("native session failed or exceeded its reserved eight-logical-turn bound")
    return session


def validate_retry_events(value: object) -> str:
    if isinstance(value, str) and value in {"none", "unknown"}:
        return value
    if isinstance(value, str) and re.fullmatch(r"visible-retries:[0-9]+", value):
        return value
    raise ValueError("retry observations must be 'none', 'unknown', or visible-retries:<count>; raw logs are not accepted")


def expected_case_before(output: Path, manifest: dict[str, object], channel: str, case: str) -> object:
    if case == "rerun":
        verified = output / f"evidence/live/{channel}/fresh/verified.json"
        if not verified.is_file() or read_json(verified).get("outcome") != "PASS":
            raise ValueError(f"{channel} rerun requires an independently verified fresh after-snapshot")
        return read_json(output / f"evidence/snapshots/after/{channel}-fresh.json")
    relative = manifest["beforeSnapshots"][f"{channel}/{case}"]
    return read_json(output / relative)


def verify_created_outputs(apply: object, digest: str, after: object) -> None:
    if not isinstance(apply, dict) or apply.get("status") != "applied" or apply.get("planDigest") != digest:
        raise ValueError("successful adoption requires an applied result bound to the reviewed plan")
    if sorted(apply_paths(apply, "created")) != sorted(APPROVED_OUTPUTS):
        raise ValueError("successful adoption must report exactly the four approved creates")
    entries = snapshot_index(after)
    for path, expected_hash in OUTPUT_HASHES.items():
        expected_bytes = (GREENFIELD / "expected" / path).read_bytes()
        entry = entries.get(path)
        if not entry or entry.get("type") != "file" or entry.get("sha256") != expected_hash or entry.get("base64") != base64.b64encode(expected_bytes).decode("ascii"):
            raise ValueError(f"successful adoption output differs from independently authored bytes: {path}")


def command_verify_outcome(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    case = args.case
    manifest = load_manifest(output)
    profile_name = manifest_profile_name(manifest)
    if case not in authorization_profile(profile_name)["verifyCases"]:
        raise ValueError(f"root case {case!r} is outside the {profile_name} verification profile")
    result_path = output / args.result
    before_path = output / args.before
    after_path = output / args.after
    if result_path != output / "evidence/live/root-cli" / case / "result.json":
        raise ValueError("result path is not this channel/case's canonical evidence slot")
    if after_path != output / "evidence/snapshots/after" / f"root-cli-{case}.json":
        raise ValueError("after snapshot path is not this channel/case's canonical evidence slot")
    if before_path != output / "evidence/snapshots/run-before" / f"root-cli-{case}.json":
        raise ValueError("before snapshot path is not this channel/case's captured launch state")
    result = read_json(result_path)
    before = read_json(before_path)
    after = read_json(after_path)
    if not isinstance(result, dict) or result.get("schemaVersion") != 2 or result.get("channel") != "root-cli" or result.get("case") != case:
        raise ValueError("root result is missing its bound schema/channel/case")
    attempt = read_json(result_path.parent / "attempt.json")
    if not isinstance(attempt, dict) or attempt.get("case") != case or attempt.get("rawInputStored") is not False or attempt.get("rawStreamsStored") is not False:
        raise ValueError("attempt evidence is missing, mismatched, or retains raw input/streams")
    if attempt.get("exitCode") != result.get("exitCode") or attempt.get("applicationRequestOutcome") != result.get("applicationRequestOutcome"):
        raise ValueError("attempt/result settlement fields disagree")
    target = target_for(output, manifest, "root-cli", case)
    if snapshot_tree(target) != after:
        raise ValueError("after snapshot no longer matches the actual target tree")
    if before != expected_case_before(output, manifest, "root-cli", case):
        raise ValueError("root before snapshot differs from the independently pinned fixture/fresh state")
    if result.get("beforeSnapshotSha256") != sha256(stable_json(before)):
        raise ValueError("result is not bound to the supplied complete before snapshot")
    if result.get("afterSnapshot") != str(after_path.relative_to(output)):
        raise ValueError("result after-snapshot path differs from the supplied snapshot")

    if case == "noninteractive":
        if result.get("exitCode") != 3 or result.get("errorCode") != "refusal.interactive-required":
            raise ValueError("noninteractive control did not preserve its actual early refusal exit/code")
        if result.get("status") is not None or result.get("planDigest") is not None or result.get("logicalProviderCalls") != 0:
            raise ValueError("noninteractive control must be an explicit zero-call no-plan result")
        if (result_path.parent / "reviewed-plan.json").exists() or (result_path.parent / "prelaunch-plan.json").exists():
            raise ValueError("noninteractive branch unexpectedly contains a plan artifact")
        problems = verify_snapshot_delta(before, after, {})
        if problems:
            raise ValueError("noninteractive snapshot changed:\n- " + "\n- ".join(problems))
        verified = {"schemaVersion": 1, "case": case, "channel": "root-cli", "outcome": "PASS", "plan": "none-by-contract", "exitCode": 3}
        atomic_json(result_path.parent / "verified.json", verified)
        print(json.dumps(verified, sort_keys=True))
        return

    plan_path = result_path.parent / "reviewed-plan.json"
    prelaunch_path = result_path.parent / "prelaunch-plan.json"
    if not plan_path.is_file() or not prelaunch_path.is_file():
        raise ValueError("complete prelaunch and post-run root plans are both required")
    plan, digest = verify_plan_shape(read_json(plan_path), target, case)
    validate_case_plan(plan, case)
    prelaunch, prelaunch_digest = verify_plan_shape(read_json(prelaunch_path), target, case)
    if prelaunch_digest != digest or prelaunch != plan or result.get("planDigest") != digest:
        raise ValueError("root result and independent prelaunch plan do not bind to the same complete digest-valid plan")
    if result.get("actualWorkflowDiagnostics") != "matched-independent-prelaunch-probes":
        raise ValueError("root result does not retain validated inventory/validation from the actual adopt workflow")

    expected_flow_source = "override" if case == "override" else "embedded"
    expected_flow_hash = manifest["flowAsset"]["overrideSha256"] if case == "override" else manifest["flowAsset"]["sha256"]
    flow_asset = result.get("flowAsset")
    if not isinstance(flow_asset, dict) or flow_asset.get("source") != expected_flow_source or flow_asset.get("sha256") != expected_flow_hash:
        raise ValueError("root actual result did not use the expected per-case live-flow source/hash")
    validate_embedded_tool_diagnostics(result.get("inventory"), result.get("validation"), result.get("diagnosticExits"), before, case)
    checkpoint_path = output / "evidence/human-checkpoints/root-cli" / f"{case}.json"
    checkpoint = read_human_checkpoint(checkpoint_path, "root-cli", case, digest)
    check_human_decision(checkpoint, case, digest)

    apply = result.get("apply")
    expected_exit = expected_operation_exit(case)
    if result.get("exitCode") != expected_exit:
        raise ValueError(f"actual root process exit {result.get('exitCode')!r} differs from expected {expected_exit}")
    if case in ROOT_CALL_CASES:
        if result.get("provider") != "OpenCode Go" or result.get("model") != "deepseek-v4-flash" or result.get("endpoint") != "https://opencode.ai/zen/go/v1/chat/completions":
            raise ValueError("provider/model/endpoint differ from the authorized root CLI route")
        if result.get("logicalProviderCalls") != 1:
            raise ValueError("eligible root case lacks exactly one observed application-level response")
        ledger = ledger_read(output / "evidence/ledger.json")
        validate_ledger_profile_binding(manifest, ledger)
        ledger_validate(ledger)
        entries = [row for row in ledger["reservedRootCalls"] if row.get("case") == case]
        if len(entries) != 1 or entries[0].get("attemptState") != "consumed" or entries[0].get("attemptOutcome") != "completed" or entries[0].get("logicalChatRequests") != 1:
            raise ValueError("root result is not bound to exactly one consumed, settled ledger reservation")
        if entries[0].get("planDigest") != digest or entries[0].get("beforeSnapshotSha256") != result.get("beforeSnapshotSha256"):
            raise ValueError("ledger reservation does not bind to this plan and complete before snapshot")
    elif result.get("logicalProviderCalls") != 0:
        raise ValueError("veto case unexpectedly reached the provider")

    if case in {"fresh", "partial-engineering", "override"}:
        if result.get("status") != "applied":
            raise ValueError("eligible create case did not report applied")
        verify_created_outputs(apply, digest, after)
        additions = {path: "file" for path in APPROVED_OUTPUTS}
        additions.update({path: "directory" for path in expected_added_directories(APPROVED_OUTPUTS, before)})
        problems = verify_snapshot_delta(before, after, additions)
    elif case == "rerun":
        if result.get("status") != "current" or not isinstance(apply, dict) or apply.get("status") != "current" or apply.get("planDigest") != digest:
            raise ValueError("identical rerun is not a current result bound to its newly reviewed plan")
        if apply_paths(apply, "created") or sorted(apply_paths(apply, "identicalUntouched")) != sorted(APPROVED_OUTPUTS):
            raise ValueError("identical rerun must create nothing and classify all four outputs untouched")
        fresh_after = read_json(output / "evidence/snapshots/after/root-cli-fresh.json")
        if before != fresh_after:
            raise ValueError("rerun before snapshot is not the independently verified fresh after snapshot")
        problems = verify_snapshot_delta(before, after, {})
    elif case == "decline":
        if result.get("status") != "refused" or apply is not None:
            raise ValueError("decline must refuse with apply:null and no output writes")
        problems = verify_snapshot_delta(before, after, {})
    elif case in {"collision", "partial-legacy"}:
        if result.get("status") != "refused" or apply is not None or not plan.get("veto"):
            raise ValueError("pre-provider veto must preserve a full veto plan and apply:null")
        problems = verify_snapshot_delta(before, after, {})
    else:
        if result.get("status") != "refused" or not isinstance(apply, dict) or apply.get("status") != "refused" or apply.get("planDigest") != digest:
            raise ValueError("stale apply did not return a refused result bound to its formerly reviewed plan")
        if apply_paths(apply, "created"):
            raise ValueError("stale apply created a file")
        collision_hash = OVERLAYS["collision"][2]
        collision_bytes = (PRESERVATION / "output-collision/docs/reference/usage.md").read_bytes()
        entry = snapshot_index(after).get("docs/reference/usage.md")
        if not entry or entry.get("type") != "file" or entry.get("sha256") != collision_hash or entry.get("base64") != base64.b64encode(collision_bytes).decode("ascii"):
            raise ValueError("stale controller collision is not preserved byte-for-byte")
        event_path = output / "evidence/events/stale-controller-root-cli.json"
        event = read_json(event_path)
        expected_event_before = str(Path(manifest["beforeSnapshots"]["root-cli/stale"]))
        expected_event_after = str((output / "evidence/snapshots" / "stale-after-controller-root-cli.json").relative_to(output))
        if not isinstance(event, dict) or event.get("channel") != "root-cli" or event.get("controllerStatus") != "inserted" or event.get("approvalAllowed") is not True or event.get("reviewedPlanDigest") != digest or event.get("planArtifactSha256") != sha256(stable_json(plan)) or event.get("beforeSnapshot") != expected_event_before or event.get("afterSnapshot") != expected_event_after or event.get("insertedPath") != "docs/reference/usage.md" or event.get("insertedSha256") != collision_hash or event.get("controllerGeneratedOrForwardedApproval") is not False or read_json(output / expected_event_after) != after:
            raise ValueError("root stale-controller event does not bind the reviewed digest and exact collision")
        additions = {"docs": "directory", "docs/reference": "directory", "docs/reference/usage.md": "file"}
        problems = verify_snapshot_delta(before, after, additions)

    if problems:
        raise ValueError("root full-snapshot verification failed:\n- " + "\n- ".join(problems))
    verified = {"schemaVersion": 1, "case": case, "outcome": "PASS", "planDigest": digest, "exitCode": expected_exit, "snapshot": "full path/type/mode/byte comparison passed"}
    atomic_json(result_path.parent / "verified.json", verified)
    print(json.dumps(verified, sort_keys=True))


def verify_control_evidence(output: Path, channel: str) -> dict[str, object]:
    manifest = load_manifest(output)
    evidence_dir = output / "evidence/live" / channel / "noninteractive"
    result_path = evidence_dir / "result.json"
    before_path = output / "evidence/snapshots/run-before" / f"{channel}-noninteractive.json"
    after_path = output / "evidence/snapshots/after" / f"{channel}-noninteractive.json"
    result = read_json(result_path)
    attempt = read_json(evidence_dir / "attempt.json")
    before = read_json(before_path)
    after = read_json(after_path)
    target = target_for(output, manifest, channel, "noninteractive")
    expected_outcome = "not-started" if channel == "skill" else None
    if not isinstance(result, dict) or result.get("schemaVersion") != 2 or result.get("channel") != channel or result.get("case") != "noninteractive" or result.get("status") is not None or result.get("outcome") != expected_outcome or result.get("planDigest") is not None:
        raise ValueError("noninteractive control result is not channel-bound or unexpectedly contains a plan")
    if result.get("exitCode") != 3 or result.get("operationExitCode") != 3 or result.get("sessionExitCode") is not None or result.get("errorCode") != "refusal.interactive-required" or result.get("logicalProviderCalls") != 0 or result.get("stdoutBytes") != 0:
        raise ValueError("noninteractive control did not preserve its exact early zero-call refusal")
    if not isinstance(attempt, dict) or attempt.get("channel") != channel or attempt.get("case") != "noninteractive" or attempt.get("exitCode") != 3 or attempt.get("applicationRequestOutcome") != 0 or attempt.get("rawInputStored") is not False or attempt.get("rawStreamsStored") is not False:
        raise ValueError("noninteractive attempt record is incomplete or disagrees with the actual result")
    if snapshot_tree(target) != after or before != expected_case_before(output, manifest, channel, "noninteractive") or before != after:
        raise ValueError("noninteractive full before/after snapshots differ or are not the pinned fixture")
    if result.get("beforeSnapshotSha256") != sha256(stable_json(before)) or result.get("afterSnapshot") != str(after_path.relative_to(output)):
        raise ValueError("noninteractive result is not bound to its complete canonical snapshots")
    if (evidence_dir / "prelaunch-plan.json").exists() or (evidence_dir / "reviewed-plan.json").exists():
        raise ValueError("noninteractive refusal unexpectedly retained a plan artifact")
    return {"schemaVersion": 1, "case": "noninteractive", "channel": channel, "outcome": "PASS", "plan": "none-by-contract", "exitCode": 3}


def command_verify_control(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    if "noninteractive" not in authorization_profile(manifest_profile_name(manifest))["verifyCases"]:
        raise ValueError("noninteractive control is outside this preparation's verification profile")
    verified = verify_control_evidence(output, args.channel)
    evidence_dir = output / "evidence/live" / args.channel / "noninteractive"
    atomic_json(evidence_dir / "verified.json", verified)
    print(json.dumps(verified, sort_keys=True))


def package_root_control_cases(output: Path, manifest: dict[str, object], offline_only: bool) -> list[dict[str, object]]:
    expected = {} if offline_only else {
        "collision": ("refusal.output-collision", "docs/reference/usage.md"),
        "partial-legacy": ("refusal.legacy-docs", "docs/"),
        "noninteractive": None,
    }
    cases: list[dict[str, object]] = []
    for case, finding in expected.items():
        evidence_dir = output / "evidence/live/root-cli" / case
        result_path = evidence_dir / "result.json"
        before_path = output / manifest["beforeSnapshots"][f"root-cli/{case}"]
        after_path = output / "evidence/snapshots/after" / f"root-cli-{case}.json"
        result = read_json(result_path)
        before = read_json(before_path)
        after = read_json(after_path)
        if not isinstance(result, dict) or result.get("schemaVersion") != 2 or result.get("channel") != "root-cli" or result.get("case") != case:
            raise ValueError(f"{case} zero-inference control result does not match its schema/channel/case")
        case_evidence: dict[str, object] = {
            "case": case,
            "result": result,
            "resultSha256": sha256(stable_json(result)),
            "beforeSnapshot": before,
            "beforeSnapshotSha256": sha256(stable_json(before)),
            "afterSnapshot": after,
            "afterSnapshotSha256": sha256(stable_json(after)),
        }
        if case == "noninteractive":
            verified = verify_control_evidence(output, "root-cli")
            stored_verified = read_json(evidence_dir / "verified.json")
            if stored_verified != verified:
                raise ValueError("noninteractive no-plan verification is missing or differs from the shared verifier")
            case_evidence["verified"] = verified
            cases.append(case_evidence)
            continue

        if result.get("exitCode") != 3 or result.get("status") != "refused" or result.get("applicationRequestOutcome") != 0 or result.get("logicalProviderCalls") != 0 or before != after:
            raise ValueError(f"{case} pre-provider veto did not prove zero logical calls and unchanged target bytes")
        plan_path = evidence_dir / "reviewed-plan.json"
        plan = read_json(plan_path)
        target = target_for(output, manifest, "root-cli", case)
        plan, digest = verify_plan_shape(plan, target, case)
        validate_case_plan(plan, case)
        if result.get("planDigest") != digest or not plan.get("veto") or result.get("apply") is not None:
            raise ValueError(f"{case} result does not bind the complete vetoed plan or unexpectedly contains an apply result")
        findings = plan.get("findings")
        if not isinstance(findings, list) or len(findings) != 1 or not isinstance(findings[0], dict) or (findings[0].get("code"), findings[0].get("path")) != finding:
            raise ValueError(f"{case} complete plan lacks its exact independently expected refusal finding")
        flow_asset = result.get("flowAsset")
        if not isinstance(flow_asset, dict) or flow_asset.get("source") != "embedded" or flow_asset.get("sha256") != manifest["flowAsset"]["sha256"]:
            raise ValueError(f"{case} no-inference result did not use its case-local live-flow source/hash")
        validate_embedded_tool_diagnostics(result.get("inventory"), result.get("validation"), result.get("diagnosticExits"), before, case)
        checkpoint_path = output / "evidence/human-checkpoints/root-cli" / f"{case}.json"
        checkpoint = read_human_checkpoint(checkpoint_path, "root-cli", case, digest)
        check_human_decision(checkpoint, case, digest)
        verified = read_json(evidence_dir / "verified.json")
        if not isinstance(verified, dict) or verified.get("outcome") != "PASS" or verified.get("case") != case or verified.get("planDigest") != digest:
            raise ValueError(f"{case} owner-reviewed root verification is not PASS for this exact plan")
        classifications = [item.get("classification") for item in plan.get("outputs", [])]
        if (case == "collision" and classifications.count("different-collision") != 1) or (case == "partial-legacy" and classifications.count("refusal") != 4):
            raise ValueError(f"{case} full plan did not preserve the expected all-create veto")
        case_evidence["reviewedPlan"] = plan
        case_evidence["reviewedPlanSha256"] = sha256(stable_json(plan))
        case_evidence["independentlyCheckedPlanDigest"] = digest
        case_evidence["verified"] = verified
        cases.append(case_evidence)
    return cases


def actual_root_veto_pack_regression(binary: Path) -> dict[str, object]:
    with tempfile.TemporaryDirectory(prefix="lds-bds251-s10-root-veto-") as temporary:
        output = Path(temporary) / "prepared"
        with redirect_stdout(io.StringIO()):
            prepare(output)
        override_bytes = (output / "evidence/override-seed/adopt.md").read_bytes()
        for channel in ("root-cli", "skill"):
            override_path = output / "xdg" / channel / "override/lds/flows/adopt.md"
            override_path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
            override_path.write_bytes(override_bytes)
        with redirect_stdout(io.StringIO()):
            command_pin_runtime(argparse.Namespace(prepared=str(output), binary=str(binary)))
        manifest = load_manifest(output)
        helper = Path(__file__).resolve()
        environment = safe_environment("OFFLINE_NO_PROVIDER_REQUEST", output / "xdg/root-cli/noninteractive", output / "home")

        def run_direct_terminal(case: str) -> bytes:
            master_fd, slave_fd = os.openpty()
            process = subprocess.Popen(
                [sys.executable, str(helper), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", case],
                stdin=slave_fd, stdout=slave_fd, stderr=slave_fd, env=environment, cwd=ROOT, start_new_session=True,
            )
            os.close(slave_fd)
            try:
                process.wait(timeout=20)
                rendered = bytearray()
                while True:
                    ready, _, _ = select.select([master_fd], [], [], 0.05)
                    if not ready:
                        break
                    try:
                        rendered.extend(os.read(master_fd, 8192))
                    except OSError:
                        break
                if process.returncode != 0:
                    raise ValueError(f"actual baseline {case} direct-terminal helper exited {process.returncode}: {bytes(rendered[-1200:])!r}")
                return bytes(rendered)
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
                os.close(master_fd)

        for case in ("collision", "partial-legacy"):
            terminal = run_direct_terminal(case)
            evidence_dir = output / "evidence/live/root-cli" / case
            result = read_json(evidence_dir / "result.json")
            before = read_json(output / manifest["beforeSnapshots"][f"root-cli/{case}"])
            after = read_json(output / "evidence/snapshots/after" / f"root-cli-{case}.json")
            if result.get("exitCode") != 3 or result.get("status") != "refused" or result.get("logicalProviderCalls") != 0 or result.get("apply") is not None or before != after:
                raise ValueError(f"actual baseline {case} did not retain exit-3 pre-provider veto and unchanged snapshot")
            if result.get("actualWorkflowDiagnostics") != "matched-independent-prelaunch-probes":
                raise ValueError(f"actual baseline {case} embedded diagnostic bodies did not match standalone probes")
            plan = read_json(evidence_dir / "reviewed-plan.json")
            plan, digest = verify_plan_shape(plan, target_for(output, manifest, "root-cli", case), case)
            if not plan.get("veto") or "Complete deterministic plan" not in terminal.decode("utf-8", errors="replace"):
                raise ValueError(f"actual baseline {case} did not display its complete veto plan in the direct terminal")
            checkpoint_path = output / "evidence/human-checkpoints/root-cli" / f"{case}.json"
            atomic_json(checkpoint_path, {
                "schemaVersion": 1, "testOnly": True, "synthetic": True, "ownerAuthored": True,
                "channel": "root-cli", "case": case, "providerConsent": "not-requested",
                "providerConsentPersonallyTyped": False, "reviewedPlanDigest": digest,
                "fullPlanPersonallyReviewed": True, "decision": "veto-no-confirmation",
                "decisionPersonallyTyped": False, "typedDecision": "not-applicable",
            })
            with redirect_stdout(io.StringIO()):
                command_verify_outcome(argparse.Namespace(
                    prepared=str(output), case=case,
                    result=f"evidence/live/root-cli/{case}/result.json",
                    before=f"evidence/snapshots/run-before/root-cli-{case}.json",
                    after=f"evidence/snapshots/after/root-cli-{case}.json",
                ))

        noninteractive = subprocess.run(
            [sys.executable, str(helper), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", "noninteractive"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=environment, cwd=ROOT, check=False,
        )
        if noninteractive.returncode != 0:
            raise ValueError("actual baseline noninteractive no-plan root control did not settle")
        with redirect_stdout(io.StringIO()):
            command_verify_control(argparse.Namespace(prepared=str(output), channel="root-cli"))
        packaged = package_root_control_cases(output, manifest, offline_only=False)
        if [item["case"] for item in packaged] != ["collision", "partial-legacy", "noninteractive"]:
            raise ValueError("actual baseline veto pack regression did not package both vetoes and the no-plan control")
        return {
            "cases": [item["case"] for item in packaged],
            "providerRequests": 0,
            "nativeSessions": 0,
            "checkpoints": "synthetic test-only veto facts; not HUMAN-A and not rootControls",
            "outcome": "PASS: actual baseline exit-3 vetoes, full terminal plans, embedded diagnostic schemas, unchanged snapshots, and explicit no-plan control packaged",
        }


def product_mock_approval_regression() -> dict[str, object]:
    nix = shutil.which("nix")
    if not nix:
        raise ValueError("cannot run the actual-product intercepted-transport PTY regression without the repository Nix Go toolchain")
    with tempfile.TemporaryDirectory(prefix="lds-bds251-s10-go-test-") as temporary:
        home = Path(temporary) / "home"
        xdg = Path(temporary) / "xdg"
        home.mkdir(mode=0o700)
        xdg.mkdir(mode=0o700)
        command = [
            nix, "develop", "--command", "go", "test", "-count=1",
            "-run", "^TestCommandApprovalInputStartsFreshPTYEpoch$",
            "./internal/verbs/adoptflow",
        ]
        completed = subprocess.run(
            command, cwd=ROOT, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env=safe_environment("OFFLINE_DUMMY_ONLY", xdg, home), check=False,
        )
        if completed.returncode != 0:
            raise ValueError("actual-product intercepted-transport PTY regression failed")
        output = (completed.stdout + completed.stderr).decode("utf-8", errors="replace")
        if "internal/verbs/adoptflow" not in output or "ok" not in output:
            raise ValueError("actual-product intercepted-transport PTY regression emitted no package PASS result")
        return {
            "command": "nix develop --command go test -count=1 -run '^TestCommandApprovalInputStartsFreshPTYEpoch$' ./internal/verbs/adoptflow",
            "outcome": "PASS",
            "cases": ["prefetched approval declined without writes", "fresh direct-terminal digest accepted and four independent outputs verified"],
            "providerRequests": 0,
            "nativeSessions": 0,
            "scope": "actual root adopt workflow and PTY with intercepted in-process mock transport; no external provider request or HUMAN-A evidence",
        }


def command_sanitize_skill(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    if authorization_profile(manifest_profile_name(manifest))["authorization"]["nativeSkillSessions"] == 0:
        raise ValueError("this preparation profile grants no native skill evidence collection")
    if args.case == "noninteractive":
        raise ValueError("noninteractive has no native session; use the explicit no-plan compare branch")
    ledger = ledger_read(output / "evidence/ledger.json")
    if ledger.get("nativeAuthorization", {}).get("state") != "authorized":
        raise ValueError("native evidence collection is blocked: owner N2 accounting/envelope decision is still pending")
    reservations = [row for row in ledger["nativeSessions"] if row.get("case") == args.case]
    if len(reservations) != 1 or reservations[0].get("attemptState") != "consumed" or reservations[0].get("outcome") != "completed":
        raise ValueError("skill collection requires exactly one consumed, successful, settled native-session ledger reservation")

    validate_case_xdg_matrix(output, manifest)
    target = target_for(output, manifest, "skill", args.case)
    evidence_dir = output / "evidence/live/skill" / args.case
    existing_evidence = {item.name for item in evidence_dir.iterdir()} if evidence_dir.is_dir() else set()
    if existing_evidence - {"session-observation.json"}:
        raise ValueError("skill evidence slot already exists; failed/unknown sessions cannot be retried")
    if not args.flow_result or not args.plan or not args.before:
        raise ValueError("skill collection requires the actual flow result, complete reviewed plan, and pinned before snapshot")

    staging_dir = output / "evidence/collection-staging/skill" / args.case
    staging_tool_dir = staging_dir / "tool-results"
    expected_inputs = {
        "flow result": (args.flow_result, staging_dir / "flow-result.json"),
        "plan": (args.plan, staging_dir / "reviewed-plan.json"),
        "before snapshot": (args.before, staging_dir / "before-snapshot.json"),
        "tool results": (args.tool_results, staging_tool_dir),
        "session evidence": (args.session_evidence, staging_dir / "session-evidence.json"),
        "human checkpoint": (args.human_checkpoint, staging_dir / "human-checkpoint.json"),
    }
    for label, (supplied, expected_path) in expected_inputs.items():
        if Path(os.path.abspath(Path(supplied).expanduser())) != expected_path:
            raise ValueError(f"skill {label} must come from the case-bound outside-target staging slot")
        assert_no_symlink_chain(expected_path)
    stage_entries = {item.name for item in staging_dir.iterdir()} if staging_dir.is_dir() else set()
    expected_stage_entries = {"flow-result.json", "reviewed-plan.json", "before-snapshot.json", "session-evidence.json", "human-checkpoint.json", "tool-results"}
    if stage_entries != expected_stage_entries:
        raise ValueError("case-bound skill staging contains missing or unrecognized artifacts")
    expected_tool_files = {"inventory.json", "validate.json", "render.json"}
    if args.case in {"fresh", "rerun", "partial-engineering", "stale", "override"}:
        expected_tool_files.add("apply.json")
    if {item.name for item in staging_tool_dir.iterdir()} != expected_tool_files:
        raise ValueError("case-bound skill tool staging contains missing or unrecognized operations")
    flow_result = read_json(Path(args.flow_result))
    if not isinstance(flow_result, dict) or flow_result.get("schemaVersion") != 1 or flow_result.get("operation") != "adopt" or flow_result.get("status") != "preparatory" or flow_result.get("availability") != "unavailable-for-execution":
        raise ValueError("skill flow artifact is not the actual read-only lds plumbing adopt result")
    asset = flow_result.get("asset")
    content = flow_result.get("content")
    if not isinstance(asset, dict) or asset.get("relativeName") != "flows/adopt.md" or not isinstance(content, str) or not content or sha256(content.encode("utf-8")) != asset.get("sha256"):
        raise ValueError("skill live-flow content does not match its reported asset hash")
    expected_source = "override" if args.case == "override" else "embedded"
    expected_hash = manifest["flowAsset"]["overrideSha256"] if args.case == "override" else manifest["flowAsset"]["sha256"]
    if asset.get("source") != expected_source or asset.get("sha256") != expected_hash:
        raise ValueError("live shared-flow source/hash differs from the case-local XDG expectation")

    plan, digest = verify_plan_shape(read_json(Path(args.plan)), target, args.case)
    validate_case_plan(plan, args.case)
    tool_dir = assert_no_symlink_chain(staging_tool_dir)
    if not stat.S_ISDIR(tool_dir.lstat().st_mode):
        raise ValueError("skill tool-result path is not a directory")
    tool_results: dict[str, dict[str, object]] = {}
    for filename in ("inventory.json", "validate.json", "render.json"):
        value = read_json(tool_dir / filename)
        if not isinstance(value, dict):
            raise ValueError(f"actual skill tool result {filename} is not a JSON object")
        tool_results[filename] = value
    if "apply.json" in expected_tool_files:
        tool_results["apply.json"] = read_json(tool_dir / "apply.json")

    session = validate_skill_session(
        args.case,
        read_json(Path(args.session_evidence)),
        args.operation_exit_code,
        args.session_exit_code,
        args.turns,
        args.retry_events,
        args.outcome,
    )
    if reservations[0].get("observedLogicalTurns") != args.turns or reservations[0].get("retryEvents") != args.retry_events:
        raise ValueError("settled ledger turn/retry observations differ from the retained actual session evidence")

    before_path = Path(args.before).expanduser()
    before = read_json(before_path)
    snapshot_index(before)
    if before != expected_case_before(output, manifest, "skill", args.case):
        raise ValueError("skill before snapshot differs from the independently pinned fixture/fresh state")
    apply = validate_skill_tool_results(tool_results, target, before, args.case, digest)
    validate_skill_tool_exits(args.case, session.get("toolExitCodes"))
    diagnostic_exits = {name: session["toolExitCodes"][name] for name in ("inventory", "validate")}

    checkpoint = read_json(Path(args.human_checkpoint))
    if not isinstance(checkpoint, dict) or checkpoint.get("ownerAuthored") is not True or checkpoint.get("channel") != "skill" or checkpoint.get("case") != args.case or checkpoint.get("reviewedPlanDigest") != digest:
        raise ValueError("complete owner-authored chat review checkpoint is missing or does not bind to this plan")
    if checkpoint.get("fullPlanPersonallyReviewed") is not True:
        raise ValueError("owner checkpoint must attest personal review of the complete raw plan")
    checkpoint_path = output / "evidence/human-checkpoints/skill" / f"{args.case}.json"
    if checkpoint_path.exists():
        raise ValueError("skill human checkpoint destination already exists")

    install = read_json(output / "evidence/skill-install.json")
    doctor = read_json(output / "evidence/skill-doctor.json")
    skill_root = output / "home/.claude/skills/lds"
    if not isinstance(install, dict) or install.get("status") != "installed" or not isinstance(doctor, dict) or doctor.get("findings") != [] or not doctor.get("targets") or not all(item.get("state") == "current" for item in doctor["targets"]):
        raise ValueError("actual isolated installed skill is missing or not current")
    skill_hash = sha256((skill_root / "SKILL.md").read_bytes())
    preflight = read_json(ROOT / "engineering/product/bds-251-stage-4-step10-preflight-evidence.json")
    pinned = preflight.get("pinnedClaude") if isinstance(preflight, dict) else None
    if not isinstance(pinned, dict) or pinned.get("version") != "2.1.286" or pinned.get("skillMarkdownSha256") != skill_hash or session.get("installedSkillSha256") != skill_hash:
        raise ValueError("installed /lds skill bytes differ from offline pins or session evidence")

    after_path = output / "evidence/snapshots/after" / f"skill-{args.case}.json"
    after = snapshot_tree(target)
    evidence_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
    checkpoint_path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    atomic_json(checkpoint_path, checkpoint)
    atomic_json(evidence_dir / "flow-result.json", flow_result)
    atomic_json(evidence_dir / "reviewed-plan.json", plan)
    for filename, value in tool_results.items():
        atomic_json(evidence_dir / "tool-results" / filename, value)
    atomic_json(evidence_dir / "session-evidence.json", session)
    atomic_json(evidence_dir / "before-snapshot.json", before)
    write_snapshot(target, after_path)
    summary = {
        "schemaVersion": 2,
        "channel": "skill",
        "case": args.case,
        "collectionStatus": "collected-not-verified",
        "claudeCodeVersion": session["claudeCodeVersion"],
        "provider": session["provider"],
        "model": session["model"],
        "outcome": session.get("outcome"),
        "operationExitCode": session["operationExitCode"],
        "sessionExitCode": session["sessionExitCode"],
        "logicalModelTurns": session["logicalModelTurns"],
        "visibleRetryEvents": session.get("visibleRetryEvents", "not reported"),
        "wireRequestCount": "not measured; not a proven cap",
        "flowAsset": {key: asset.get(key) for key in ("relativeName", "source", "sha256")},
        "inventory": tool_results["inventory.json"],
        "validation": tool_results["validate.json"],
        "diagnosticExits": diagnostic_exits,
        "planDigest": digest,
        "reviewedPlanCanonicalSha256": sha256(stable_json(plan)),
        "planEvidenceSource": "reviewed-plan.json; single captured docs plan result bound by the observed plan process exit",
        "apply": apply,
        "targetCase": str(target.relative_to(output)),
        "beforeSnapshot": str((evidence_dir / "before-snapshot.json").relative_to(output)),
        "afterSnapshot": str(after_path.relative_to(output)),
        "humanCheckpoint": str(checkpoint_path.relative_to(output)),
        "ledgerCase": args.case,
        "installedSkillSha256": skill_hash,
    }
    atomic_json(evidence_dir / "result.json", summary)
    print(json.dumps({"case": args.case, "collectionStatus": "collected-not-verified", "result": str((evidence_dir / 'result.json').relative_to(output)), "verification": "not run; use verify-skill"}, sort_keys=True))


def skill_expected_decision(case: str) -> tuple[str, str]:
    if case in {"collision", "partial-legacy"}:
        return "veto-no-confirmation", "not-applicable"
    if case == "decline":
        return "decline", "decline"
    return "apply", "apply-digest"


def command_verify_skill(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    if authorization_profile(manifest_profile_name(manifest))["authorization"]["nativeSkillSessions"] == 0:
        raise ValueError("this preparation profile grants no native skill verification")
    validate_case_xdg_matrix(output, manifest)
    case = args.case
    if case == "noninteractive":
        raise ValueError("native noninteractive has no skill session; use the explicit no-plan compare branch")
    evidence_dir = output / "evidence/live/skill" / case
    result = read_json(evidence_dir / "result.json")
    session = read_json(evidence_dir / "session-evidence.json")
    flow = read_json(evidence_dir / "flow-result.json")
    target = target_for(output, manifest, "skill", case)
    plan, digest = verify_plan_shape(read_json(evidence_dir / "reviewed-plan.json"), target, case)
    validate_case_plan(plan, case)
    if not isinstance(result, dict) or result.get("schemaVersion") != 2 or result.get("collectionStatus") != "collected-not-verified" or result.get("case") != case:
        raise ValueError("skill collection is missing, already altered, or case-unbound")
    if result.get("planDigest") != digest or not isinstance(session, dict) or session.get("case") != case:
        raise ValueError("skill result/session evidence does not bind to the complete plan and case")
    if result.get("reviewedPlanCanonicalSha256") != sha256(stable_json(plan)) or result.get("planEvidenceSource") != "reviewed-plan.json; single captured docs plan result bound by the observed plan process exit":
        raise ValueError("skill result does not preserve the sole actual reviewed-plan artifact/provenance")
    validate_skill_session(
        case,
        session,
        result.get("operationExitCode"),
        result.get("sessionExitCode"),
        result.get("logicalModelTurns"),
        result.get("visibleRetryEvents"),
        result.get("outcome"),
    )
    if result.get("sessionExitCode") != 0 or session.get("sessionExitCode") != 0:
        raise ValueError("Claude Code session exited nonzero; native outcome remains failed")
    expected_exit = expected_operation_exit(case)
    if result.get("operationExitCode") != expected_exit or session.get("operationExitCode") != expected_exit:
        raise ValueError("actual adoption operation exit differs from its expected case result")
    turns = result.get("logicalModelTurns")
    if not isinstance(turns, int) or turns < 1 or turns > 8 or session.get("logicalModelTurns") != turns:
        raise ValueError("native session turn count is absent or exceeds its reserved eight-turn ceiling")
    validate_skill_tool_exits(case, session.get("toolExitCodes"))
    if session.get("claudeCodeVersion") != "2.1.286" or session.get("provider") != "OpenCode Go Anthropic Messages compatibility gateway" or session.get("model") != "deepseek-v4-flash":
        raise ValueError("native session drifted from the version/provider/model pin")
    if session.get("skillInvocation") != "/lds" or session.get("manualPermissionReview") is not True:
        raise ValueError("session lacks actual installed skill invocation or manually reviewed permissions")
    if not isinstance(session.get("visibleRetryEvents"), (str, list)):
        raise ValueError("visible retry observations are missing; wire HTTP count remains explicitly unmeasured")

    if not isinstance(flow, dict) or flow.get("schemaVersion") != 1 or flow.get("operation") != "adopt" or flow.get("status") != "preparatory" or flow.get("availability") != "unavailable-for-execution":
        raise ValueError("actual installed-skill live-reader result is missing or malformed")
    asset = flow.get("asset")
    expected_source = "override" if case == "override" else "embedded"
    expected_hash = manifest["flowAsset"]["overrideSha256"] if case == "override" else manifest["flowAsset"]["sha256"]
    content = flow.get("content")
    if not isinstance(asset, dict) or asset.get("relativeName") != "flows/adopt.md" or asset.get("source") != expected_source or asset.get("sha256") != expected_hash or not isinstance(content, str) or sha256(content.encode("utf-8")) != expected_hash:
        raise ValueError("live shared-flow reader did not observe the expected per-case source/hash")
    before = read_json(evidence_dir / "before-snapshot.json")
    if before != expected_case_before(output, manifest, "skill", case):
        raise ValueError("skill before snapshot differs from the independently pinned fixture/fresh state")
    tool_results = {
        name: read_json(evidence_dir / "tool-results" / name)
        for name in ("inventory.json", "validate.json", "render.json", "apply.json")
        if (evidence_dir / "tool-results" / name).is_file()
    }
    verified_apply = validate_skill_tool_results(tool_results, target, before, case, digest)
    validate_skill_tool_exits(case, session.get("toolExitCodes"))
    expected_diagnostic_exits = {name: session["toolExitCodes"][name] for name in ("inventory", "validate")}
    if result.get("inventory") != tool_results.get("inventory.json") or result.get("validation") != tool_results.get("validate.json") or result.get("diagnosticExits") != expected_diagnostic_exits:
        raise ValueError("skill semantic diagnostics differ from the actual linked tool artifacts/exits")
    if result.get("apply") != verified_apply:
        raise ValueError("skill summary apply result differs from the actual linked docs apply tool result")
    install = read_json(output / "evidence/skill-install.json")
    doctor = read_json(output / "evidence/skill-doctor.json")
    skill_root = output / "home/.claude/skills/lds"
    installed_hash = sha256((skill_root / "SKILL.md").read_bytes())
    stamp_hash = sha256((skill_root / ".lds-manifest-stamp.json").read_bytes())
    installed_tree_hash = sha256(stable_json(snapshot_tree(skill_root)))
    preflight = read_json(ROOT / "engineering/product/bds-251-stage-4-step10-preflight-evidence.json")
    pinned = preflight.get("pinnedClaude") if isinstance(preflight, dict) else None
    if not isinstance(install, dict) or install.get("status") != "installed" or not isinstance(doctor, dict) or doctor.get("findings") != [] or not doctor.get("targets") or not all(item.get("state") == "current" for item in doctor["targets"]):
        raise ValueError("actual installed skill doctor/install evidence is missing or stale")
    if not isinstance(pinned, dict) or pinned.get("version") != "2.1.286" or pinned.get("skillMarkdownSha256") != installed_hash or pinned.get("manifestStampSha256") != stamp_hash or pinned.get("skillTreeSnapshotSha256") != installed_tree_hash or result.get("installedSkillSha256") != installed_hash or session.get("installedSkillSha256") != installed_hash:
        raise ValueError("installed Claude skill bytes differ from pinned, session, or collection evidence")

    ledger = ledger_read(output / "evidence/ledger.json")
    rows = [row for row in ledger["nativeSessions"] if row.get("case") == case]
    if ledger.get("nativeAuthorization", {}).get("state") != "authorized" or len(rows) != 1 or rows[0].get("attemptState") != "consumed" or rows[0].get("outcome") != "completed":
        raise ValueError("skill result is not bound to one authorized, consumed and settled ledger session")
    if rows[0].get("observedLogicalTurns") != turns or rows[0].get("retryEvents") != session.get("visibleRetryEvents"):
        raise ValueError("native ledger turn/retry observations differ from session evidence")
    session_observation = read_json(evidence_dir / "session-observation.json")
    if not isinstance(session_observation, dict) or session_observation.get("status") != "completed" or session_observation.get("sessionExitCode") != session.get("sessionExitCode") or session_observation.get("operationExitCode") != session.get("operationExitCode") or session_observation.get("logicalModelTurns") != turns or session_observation.get("visibleRetryEvents") != session.get("visibleRetryEvents") or session_observation.get("installedSkillSha256") != installed_hash:
        raise ValueError("sanitized settled native-session observation is missing or differs from the actual session evidence")

    checkpoint = read_json(output / "evidence/human-checkpoints/skill" / f"{case}.json")
    if not isinstance(checkpoint, dict) or checkpoint.get("ownerAuthored") is not True or checkpoint.get("channel") != "skill" or checkpoint.get("case") != case or checkpoint.get("reviewedPlanDigest") != digest or checkpoint.get("fullPlanPersonallyReviewed") is not True:
        raise ValueError("owner-authored full-plan chat checkpoint is missing or not digest-bound")
    decision, typed = skill_expected_decision(case)
    expected_typed = f"apply {digest}" if typed == "apply-digest" else typed
    personally_typed = decision != "veto-no-confirmation"
    if checkpoint.get("decision") != decision or checkpoint.get("decisionPersonallyTyped") is not personally_typed or checkpoint.get("typedDecision") != expected_typed:
        raise ValueError("owner checkpoint does not preserve the personal exact-digest decision/categorical decline")

    after_path = output / result["afterSnapshot"]
    if result.get("beforeSnapshot") != str((evidence_dir / "before-snapshot.json").relative_to(output)) or result.get("targetCase") != str(target.relative_to(output)) or result.get("afterSnapshot") != str((output / "evidence/snapshots/after" / f"skill-{case}.json").relative_to(output)):
        raise ValueError("skill snapshot/target paths are not case-bound canonical evidence locations")
    after = read_json(after_path)
    if snapshot_tree(target) != after:
        raise ValueError("skill after snapshot does not match the actual target tree")
    apply = verified_apply
    if case in {"fresh", "partial-engineering", "override"}:
        if not isinstance(apply, dict) or apply.get("status") != "applied" or apply.get("planDigest") != digest or sorted(apply_paths(apply, "created")) != sorted(APPROVED_OUTPUTS):
            raise ValueError("skill success requires a digest-bound apply creating exactly the four approved outputs")
        for path, expected_hash in OUTPUT_HASHES.items():
            entry = snapshot_index(after).get(path)
            expected_bytes = (GREENFIELD / "expected" / path).read_bytes()
            if not entry or entry.get("type") != "file" or entry.get("sha256") != expected_hash or entry.get("base64") != base64.b64encode(expected_bytes).decode("ascii"):
                raise ValueError(f"skill output differs from independent expected bytes: {path}")
        additions = {path: "file" for path in APPROVED_OUTPUTS}
        additions.update({path: "directory" for path in expected_added_directories(APPROVED_OUTPUTS, before)})
    elif case == "rerun":
        if not isinstance(apply, dict) or apply.get("status") != "current" or apply.get("planDigest") != digest or apply_paths(apply, "created") or sorted(apply_paths(apply, "identicalUntouched")) != sorted(APPROVED_OUTPUTS):
            raise ValueError("skill rerun must be current, create nothing, and preserve all four outputs")
        if before != read_json(output / "evidence/snapshots/after/skill-fresh.json"):
            raise ValueError("skill rerun must use the verified fresh after-snapshot as its before-state")
        additions = {}
    elif case == "stale":
        if not isinstance(apply, dict) or apply.get("status") != "refused" or apply.get("planDigest") != digest or apply_paths(apply, "created"):
            raise ValueError("skill stale apply must refuse the old digest with zero creates")
        event = read_json(output / "evidence/events/stale-controller-skill.json")
        collision_hash = OVERLAYS["collision"][2]
        entry = snapshot_index(after).get("docs/reference/usage.md")
        expected_event_before = str(Path(manifest["beforeSnapshots"]["skill/stale"]))
        expected_event_after = str((output / "evidence/snapshots" / "stale-after-controller-skill.json").relative_to(output))
        if not isinstance(event, dict) or event.get("channel") != "skill" or event.get("controllerStatus") != "inserted" or event.get("approvalAllowed") is not True or event.get("reviewedPlanDigest") != digest or event.get("planArtifactSha256") != sha256(stable_json(plan)) or event.get("beforeSnapshot") != expected_event_before or event.get("afterSnapshot") != expected_event_after or event.get("insertedPath") != "docs/reference/usage.md" or event.get("insertedSha256") != collision_hash or event.get("controllerGeneratedOrForwardedApproval") is not False or read_json(output / expected_event_after) != after or not entry or entry.get("type") != "file" or entry.get("sha256") != collision_hash:
            raise ValueError("skill stale collision/event does not preserve the pinned old-plan race")
        additions = {"docs": "directory", "docs/reference": "directory", "docs/reference/usage.md": "file"}
    elif case == "decline":
        if result.get("outcome") != "declined" or apply is not None:
            raise ValueError("skill decline must retain no apply result and no writes")
        additions = {}
    else:
        if result.get("outcome") != "refused" or apply is not None or not plan.get("veto"):
            raise ValueError("skill collision/legacy veto must retain no apply result and no writes")
        additions = {}
    problems = verify_snapshot_delta(before, after, additions)
    if problems:
        raise ValueError("skill full-snapshot verification failed:\n- " + "\n- ".join(problems))
    result["collectionStatus"] = "verified-PASS"
    result["verification"] = "actual installed skill/live reader/tool results, human checkpoint, ledger, exits, turns, and full snapshots passed"
    atomic_json(evidence_dir / "result.json", result)
    atomic_json(evidence_dir / "verified.json", {"schemaVersion": 1, "case": case, "outcome": "PASS", "planDigest": digest})
    print(json.dumps({"case": case, "outcome": "PASS", "planDigest": digest, "sessionExitCode": 0, "logicalModelTurns": turns, "wireRequestCount": "not measured"}, sort_keys=True))


def semantic_plan(plan: dict[str, object]) -> dict[str, object]:
    projected = dict(plan)
    for key in ("targetRoot", "targetIdentity", "planDigest"):
        projected.pop(key)
    parents = projected.get("existingParentDirectories")
    if isinstance(parents, list):
        normalized = []
        for parent in parents:
            if not isinstance(parent, dict):
                raise ValueError("existing parent directory entry is malformed")
            item = dict(parent)
            item.pop("identity", None)
            normalized.append(item)
        projected["existingParentDirectories"] = normalized
    return projected


def semantic_apply(apply: object, digest: str) -> object:
    if apply is None:
        return None
    if not isinstance(apply, dict) or apply.get("planDigest") != digest or apply.get("schemaVersion") != 1 or apply.get("operation") != "docs apply":
        raise ValueError("apply result is missing or not bound to its own complete plan")
    if apply.get("status") not in {"applied", "current", "refused", "partial"}:
        raise ValueError("apply result has an unknown status")
    if not {"created", "identicalUntouched", "unattempted", "failed", "preserved"}.issubset(apply):
        raise ValueError("apply result is missing required effect fields")
    for key in ("created", "identicalUntouched", "unattempted", "preserved"):
        entries = apply.get(key)
        if not isinstance(entries, list):
            raise ValueError(f"apply result lacks its complete {key} array")
        for entry in entries:
            if not isinstance(entry, dict) or not isinstance(entry.get("path"), str):
                raise ValueError(f"apply result {key} contains an invalid path entry")
    failed = apply.get("failed")
    if failed is not None and (not isinstance(failed, dict) or not isinstance(failed.get("path"), str) or not isinstance(failed.get("code"), str)):
        raise ValueError("apply failure must be null or a path/code object")
    projected = dict(apply)
    del projected["planDigest"]
    return projected


def semantic_result(value: object, digest: str, channel: str, case: str) -> dict[str, object]:
    if not isinstance(value, dict) or value.get("planDigest") != digest:
        raise ValueError(f"{channel} result is not bound to its own complete plan")
    if channel == "root-cli":
        outcome = "declined" if case == "decline" else value.get("status")
        operation_exit = value.get("exitCode")
        session_exit = None
    else:
        outcome = value.get("outcome")
        operation_exit = value.get("operationExitCode")
        session_exit = value.get("sessionExitCode")
        if not isinstance(session_exit, int) or isinstance(session_exit, bool) or session_exit != 0:
            raise ValueError("native session exit was not zero")
        turns = value.get("logicalModelTurns")
        if not isinstance(turns, int) or isinstance(turns, bool) or turns < 1 or turns > 8:
            raise ValueError("native session did not stay within its reserved logical-turn limit")
    if not isinstance(operation_exit, int) or isinstance(operation_exit, bool) or operation_exit != expected_operation_exit(case):
        raise ValueError(f"{channel} operation exit differs from the expected case refusal/success")
    inventory = value.get("inventory")
    validation = value.get("validation")
    if channel == "skill":
        inventory = standalone_tool_body(inventory, "docs inventory", {"files", "links", "skipped"})
        validation = standalone_tool_body(validation, "docs validate", {"findings", "skipped"})
    diagnostic_exits = value.get("diagnosticExits")
    if not isinstance(inventory, dict) or not isinstance(validation, dict) or diagnostic_exits != {"inventory": 0, "validate": 1 if case == "collision" else 0}:
        raise ValueError(f"{channel} result is missing the actual inventory/validation diagnostics and exits")
    return {
        "outcome": outcome,
        "operationExitCode": operation_exit,
        "sessionExitCode": session_exit,
        "flowAsset": value.get("flowAsset"),
        "inventory": inventory,
        "validation": validation,
        "diagnosticExits": diagnostic_exits,
        "apply": semantic_apply(value.get("apply"), digest),
    }


def command_compare_semantic(args: argparse.Namespace) -> None:
    left_result = read_json(Path(args.left_result))
    right_result = read_json(Path(args.right_result))
    left_before = read_json(Path(args.left_before))
    right_before = read_json(Path(args.right_before))
    left_after = read_json(Path(args.left_after))
    right_after = read_json(Path(args.right_after))
    mismatches: list[str] = []
    if args.case == "noninteractive":
        root_control, skill_control = left_result, right_result
        if not isinstance(root_control, dict) or root_control.get("channel") != "root-cli" or root_control.get("case") != args.case or root_control.get("status") is not None or root_control.get("planDigest") is not None or root_control.get("exitCode") != 3 or root_control.get("logicalProviderCalls") != 0:
            mismatches.append("root CLI is not the explicit exit-3 zero-call noninteractive no-plan refusal")
        if not isinstance(skill_control, dict) or skill_control.get("channel") != "skill" or skill_control.get("case") != args.case or skill_control.get("outcome") != "not-started" or skill_control.get("planDigest") is not None or skill_control.get("operationExitCode") != 3 or skill_control.get("sessionExitCode") is not None or skill_control.get("logicalProviderCalls") != 0:
            mismatches.append("skill-channel control is not the explicit zero-session/zero-call no-plan refusal")
        if args.left_plan or args.right_plan:
            mismatches.append("noninteractive no-plan branch must not receive plan artifacts")
        if left_before != left_after or right_before != right_after:
            mismatches.append("noninteractive full before/after snapshots differ")
        if left_before != right_before or left_after != right_after:
            mismatches.append("noninteractive root/skill full snapshots differ")
        for result_path, channel in ((Path(args.left_result), "root-cli"), (Path(args.right_result), "skill")):
            verified = read_json(result_path.parent / "verified.json")
            if not isinstance(verified, dict) or verified.get("outcome") != "PASS" or verified.get("case") != args.case or verified.get("channel") != channel:
                mismatches.append(f"{channel} explicit no-plan verification is missing or not PASS")
        if mismatches:
            raise ValueError("noninteractive semantic comparison failed:\n- " + "\n- ".join(mismatches))
        print(json.dumps({"case": args.case, "plans": "none-by-contract", "wholeSnapshots": "equal", "outcomesAndExits": "equal"}, sort_keys=True))
        return

    if not args.left_plan or not args.right_plan:
        raise ValueError("plan-bearing comparison requires both complete plan artifacts")
    left_plan, left_digest = check_plan_digest(read_json(Path(args.left_plan)))
    right_plan, right_digest = check_plan_digest(read_json(Path(args.right_plan)))
    if not isinstance(left_result, dict) or left_result.get("case") != args.case or left_result.get("planDigest") != left_digest:
        mismatches.append("root result is not bound to the supplied complete plan/case")
    if not isinstance(right_result, dict) or right_result.get("case") != args.case or right_result.get("planDigest") != right_digest:
        mismatches.append("skill result is not bound to the supplied complete plan/case")
    for channel, case in (("root-cli", "root-cli"), ("skill", "skill")):
        verified_path = Path(args.left_result if channel == "root-cli" else args.right_result).parent / "verified.json"
        verified = read_json(verified_path)
        if not isinstance(verified, dict) or verified.get("outcome") != "PASS" or verified.get("case") != args.case:
            mismatches.append(f"{channel} independent per-case verification is not PASS")
    if semantic_plan(left_plan) != semantic_plan(right_plan):
        mismatches.append("full plans differ outside the exact target identity/digest fields")
    left_semantics = semantic_result(left_result, left_digest, "root-cli", args.case)
    right_semantics = semantic_result(right_result, right_digest, "skill", args.case)
    for channel, semantics, before in (("root-cli", left_semantics, left_before), ("skill", right_semantics, right_before)):
        try:
            validate_embedded_tool_diagnostics(
                semantics["inventory"], semantics["validation"], semantics["diagnosticExits"], before, args.case,
            )
        except ValueError as error:
            mismatches.append(f"{channel} diagnostic evidence is invalid against its complete before snapshot: {error}")
    if {key: value for key, value in left_semantics.items() if key != "sessionExitCode"} != {key: value for key, value in right_semantics.items() if key != "sessionExitCode"}:
        mismatches.append("semantic outcomes, applicable operation exits, flow sources, or apply/preservation results differ")
    if left_before != right_before:
        mismatches.append("full before snapshots differ byte-for-byte")
    if left_after != right_after:
        mismatches.append("full after snapshots differ byte-for-byte")
    if mismatches:
        raise ValueError("paired root/skill comparison failed:\n- " + "\n- ".join(mismatches))
    print(json.dumps({"case": args.case, "semanticPlans": "equal except exact target identity/digest fields", "outcomesAndExits": "equal", "wholeBeforeSnapshots": "equal", "wholeAfterSnapshots": "equal", "literalPlanDigestsCompared": False}, sort_keys=True))


def command_pack_preflight(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    manifest = load_manifest(output)
    if manifest_profile_name(manifest) != "executor":
        raise ValueError("supplemental-decline evidence is not a full executor preflight and cannot be packaged as one")
    validate_case_xdg_matrix(output, manifest)
    destination = Path(args.out).expanduser().resolve()
    if destination == ROOT or ROOT not in destination.parents:
        raise ValueError("preflight evidence output must be a new file inside this repository")
    if destination.exists() and not args.replace_existing:
        raise ValueError("refusing to overwrite preflight evidence")
    if destination.exists():
        prior = read_json(destination)
        if not isinstance(prior, dict) or prior.get("baseline") != BASELINE or prior.get("providerRequests") != 0 or prior.get("nativeSessions") != 0:
            raise ValueError("replacement is limited to the existing same-baseline zero-inference S10 preflight evidence")
    verify_baseline()
    root_cli_binary = Path(args.root_cli_binary).expanduser().resolve(strict=True)
    validate_runtime(output, manifest, root_cli_binary)
    actual_cli_checks = actual_cli_diagnostic_regressions(root_cli_binary, output, manifest)
    actual_veto_pack = actual_root_veto_pack_regression(root_cli_binary)
    product_mock_approval = product_mock_approval_regression()
    staged_skill_collection = offline_skill_collection_regression(root_cli_binary, output)
    offline_validation = offline_regressions()

    cases = package_root_control_cases(output, manifest, args.offline_only)

    install = read_json(output / "evidence/skill-install.json")
    doctor = read_json(output / "evidence/skill-doctor.json")
    skill_root = output / "home/.claude/skills/lds"
    installed_snapshot = read_json(output / "evidence/snapshots/skill-after-install.json")
    overridden_snapshot = read_json(output / "evidence/snapshots/skill-after-override.json")
    if installed_snapshot != overridden_snapshot:
        raise ValueError("adding the post-install XDG live override changed the generated installed skill tree")
    flow = read_json(output / "evidence/skill-override-flow.json")
    if not isinstance(flow, dict) or not isinstance(flow.get("asset"), dict):
        raise ValueError("isolated lds plumbing adopt override provenance is missing")
    if not isinstance(doctor, dict) or doctor.get("findings") != [] or not doctor.get("targets") or not all(item.get("state") == "current" for item in doctor.get("targets", [])):
        raise ValueError("isolated Claude skill doctor did not report the installation current")
    if not isinstance(install, dict) or install.get("status") != "installed":
        raise ValueError("isolated Claude skill installation did not report installed")
    flow_asset = flow["asset"]
    if flow_asset.get("source") != "override" or flow_asset.get("sha256") != manifest["flowAsset"]["overrideSha256"]:
        raise ValueError("post-install live override provenance/hash differs from the prepared override")
    help_text = (output / "evidence/claude-help.txt").read_text(encoding="utf-8")
    reported_version = (output / "evidence/claude-version.txt").read_text(encoding="utf-8").strip()
    expected_version_output = f"{args.claude_version} (Claude Code)"
    if reported_version != expected_version_output:
        raise ValueError(f"offline pinned Claude --version result differs from {expected_version_output!r}")
    required_flags = ("--tools", "--disallowedTools", "--permission-mode", "--strict-mcp-config", "--mcp-config", "--setting-sources")
    missing_flags = [flag for flag in required_flags if flag not in help_text]
    if missing_flags:
        raise ValueError(f"pinned Claude help does not expose required isolation flags: {missing_flags}")
    skill_snapshot_bytes = stable_json(installed_snapshot)
    claude_binary = Path(args.claude_binary).expanduser().resolve(strict=True)
    evidence = {
        "schemaVersion": 1,
        "baseline": BASELINE,
        "checkedOn": datetime.now(timezone.utc).date().isoformat(),
        "scope": "offline-only correction validation; no owner-reviewed root CLI matrix/HUMAN-A controls were run under C2" if args.offline_only else "offline correction validation plus owner-reviewed zero-inference root controls; no provider request or native session was started during this correction pass",
        "preparationArtifacts": {
            "helperSha256": sha256(Path(__file__).read_bytes()),
            "preparationGuideSha256": sha256((ROOT / "engineering/product/bds-251-stage-4-step10-preparation.md").read_bytes()),
            "frozenStage7OrReconciliationEvidenceChanged": False,
        },
        "correctionValidation": {
            "offlineOnly": True,
            "productMockApprovalRegression": product_mock_approval,
            "providerRequests": 0,
            "nativeSessions": 0,
            "offlineRegressions": offline_validation,
            "actualBaselineCliDiagnostics": actual_cli_checks,
            "actualBaselineVetoPackRegression": actual_veto_pack,
            "offlineActualToolCollection": staged_skill_collection,
        },
        "providerCredentialAvailableWhilePackaging": bool(os.environ.get("OPENCODE_API_KEY", "")),
        "providerRequests": 0,
        "nativeSessions": 0,
        "rootControls": cases,
        "historicalControlLimitations": {
            "rerunUnderCorrectedHelper": not args.offline_only,
            "caseLocalXdgIsolationProvenByTheseControls": not args.offline_only,
            "detail": "C2 holds the owner-reviewed root CLI matrix. Earlier shared-override controls are not imported. Separate actual-binary veto packing used a fresh temporary target and synthetic test-only checkpoints, not HUMAN-A or rootControls." if args.offline_only else "Current helper owner-reviewed root controls passed with their case-local live-flow provenance.",
        },
        "fixtureHashes": manifest["fixtureHashes"],
        "rootCliBinarySha256": sha256(root_cli_binary.read_bytes()),
        "pinnedClaude": {
            "version": reported_version.removesuffix(" (Claude Code)"),
            "binarySha256": sha256(claude_binary.read_bytes()),
            "skillInstall": install,
            "doctor": doctor,
            "skillTreeSnapshotSha256": sha256(skill_snapshot_bytes),
            "skillMarkdownSha256": sha256((skill_root / "SKILL.md").read_bytes()),
            "manifestStampSha256": sha256((skill_root / ".lds-manifest-stamp.json").read_bytes()),
            "availableIsolationFlags": list(required_flags),
            "generatedSkillTreeUnchangedAfterOverride": True,
            "liveFlowAsset": {key: flow["asset"].get(key) for key in ("relativeName", "source", "sha256")},
        },
        "retryResearch": {
            "pinnedVersion": args.claude_version,
            "officialDocs": [
                "https://code.claude.com/docs/en/errors#automatic-retries",
                "https://code.claude.com/docs/en/env-vars",
                "https://code.claude.com/docs/en/llm-gateway-protocol#automatic-retry-and-error-forwarding",
            ],
            "maxRetriesZero": "removes the ordinary retry budget only; current error docs describe a separately bounded stalled-stream retry outside that budget",
            "gatewayCompatibility": "the gateway guide documents request reissues after some capability rejections without establishing that every path is disabled by MAX_RETRIES=0",
            "nonstreamingFallback": "the disable flag covers one fallback class only; the documented streaming-404 exception remains",
            "wireRequestCounter": "not exposed as a guaranteed counter; interactive retry indicators are not a complete wire-level count",
            "nativeInference": "blocked pending root N2 accounting/envelope choice; lean defer; Anthropic-subscription escape hatch approved in principle only and requires a new root access/model/configuration/budget decision after retaining any gateway failure/counts",
        },
        "limitations": [
            "Offline-only packaging contains no owner-reviewed root CLI controls; actual-binary veto packing used only a fresh temporary fixture and synthetic test-only checkpoints, not HUMAN-A or rootControls." if args.offline_only else "These controls prove only disposable-fixture refusals and no-write snapshots; they do not exercise provider responses.",
            "The failed S7 exactCommand attempt remains failed/unexplained.",
            "No pilot, first-release, Anthropic-support, or universal reliability claim is supported.",
        ],
    }
    atomic_json(destination, evidence)
    print(json.dumps({"evidence": str(destination.relative_to(ROOT)), "sha256": sha256(destination.read_bytes()), "rootControls": len(cases), "providerRequests": 0, "nativeSessions": 0}, sort_keys=True))


def ledger_read(path: Path) -> dict[str, object]:
    value = read_json(path)
    if not isinstance(value, dict) or value.get("schemaVersion") != 2:
        raise ValueError("invalid S10 ledger")
    authorization_profile(value.get("wave"))
    return value


def ledger_save(path: Path, value: dict[str, object]) -> None:
    atomic_json(path, value)


def ledger_validate(ledger: dict[str, object]) -> tuple[int, int, int]:
    policy = authorization_profile(ledger.get("wave"))
    authorization = ledger["authorization"]
    root_calls = ledger["reservedRootCalls"]
    sessions = ledger["nativeSessions"]
    if not isinstance(root_calls, list) or not isinstance(sessions, list):
        raise ValueError("ledger counters must be arrays")
    expected_limits = policy["authorization"]
    if not authorization_matches(authorization, expected_limits):
        raise ValueError(f"ledger authorization differs from the {ledger['wave']} profile ceilings")
    if len(root_calls) > authorization["rootChatCalls"]:
        raise ValueError("root Chat call authorization exhausted")
    if len(sessions) > authorization["nativeSkillSessions"]:
        raise ValueError("native skill session authorization exhausted")
    root_cases = [row.get("case") for row in root_calls]
    if len(root_cases) != len(set(root_cases)) or any(case not in policy["rootCases"] for case in root_cases):
        raise ValueError("root ledger contains duplicate or unauthorized case reservations")
    for row in root_calls:
        if row.get("logicalChatRequests") != 1 or row.get("attemptState") not in {"reserved", "consumed"} or row.get("attemptOutcome") not in {"pending", "unknown", "completed", "failed-or-unknown", "stopped-unknown"}:
            raise ValueError("root ledger row is not one bounded, non-retryable application-level attempt")
        if row.get("attemptState") == "reserved" and row.get("attemptOutcome") != "pending":
            raise ValueError("unconsumed root reservation has a settled outcome")
        if row.get("attemptState") == "consumed" and row.get("attemptOutcome") == "pending":
            raise ValueError("consumed root attempt was not marked unknown before launch")
    skill_cases = [row.get("case") for row in sessions]
    if len(skill_cases) != len(set(skill_cases)) or any(case not in SKILL_CASES for case in skill_cases):
        raise ValueError("native ledger contains duplicate or unknown session cases")
    native_authorization = ledger.get("nativeAuthorization")
    if not isinstance(native_authorization, dict) or native_authorization.get("state") not in {"blocked", "authorized"}:
        raise ValueError("native authorization state is missing or malformed")
    if policy["nativeMustRemainBlocked"] and native_authorization.get("state") != "blocked":
        raise ValueError("supplemental-decline profile grants zero native authorization")
    if native_authorization["state"] == "blocked" and sessions:
        raise ValueError("native skill sessions are blocked and none may be reserved")
    if policy["nativeMustRemainBlocked"] and sessions:
        raise ValueError("supplemental-decline profile permits zero native sessions")
    total_turns = 0
    for session in sessions:
        if session.get("attemptState") not in {"reserved", "consumed"} or session.get("outcome") not in {"pending", "unknown", "completed", "failed", "stopped", "not-started"}:
            raise ValueError("native session ledger row has an invalid attempt state/outcome")
        if session.get("attemptState") == "reserved" and session.get("outcome") != "pending":
            raise ValueError("unconsumed native reservation has a settled outcome")
        if session.get("attemptState") == "consumed" and session.get("outcome") == "pending":
            raise ValueError("consumed native session was not marked unknown before launch")
        turns = session.get("observedLogicalTurns")
        if turns is not None and (not isinstance(turns, int) or turns < 0):
            raise ValueError("native session logical-turn observation must be a nonnegative integer")
        if isinstance(turns, int):
            total_turns += turns
        if session.get("maximumLogicalTurns") != authorization["logicalTurnsPerNativeSession"]:
            raise ValueError("native session row does not reserve its profile turn bound")
        retry_events = session.get("retryEvents")
        if retry_events is not None:
            validate_retry_events(retry_events)
    ceiling = len(root_calls) + authorization["logicalTurnsPerNativeSession"] * len(sessions)
    if ceiling > authorization["logicalRequestsPerWave"]:
        raise ValueError("reserved logical request ceiling exceeds the selected authorization profile")
    return len(root_calls), len(sessions), total_turns


def command_ledger_reserve_root(args: argparse.Namespace) -> None:
    path = Path(args.ledger).expanduser().resolve(strict=True)
    case = args.case
    if path.name != "ledger.json" or path.parent.name != "evidence":
        raise ValueError("root reservation requires the canonical prepared evidence/ledger.json")
    output = path.parent.parent
    with ledger_lock(path):
        manifest = load_manifest(output)
        profile_name = manifest_profile_name(manifest)
        policy = authorization_profile(profile_name)
        if case not in policy["rootCases"]:
            raise ValueError(f"root case {case!r} is not authorized by the {profile_name} profile")
        pins = read_json(output / "evidence/runtime-pins.json")
        if not isinstance(pins, dict) or not isinstance(pins.get("binaryPath"), str):
            raise ValueError("root reservation requires its immutable runtime pin")
        binary = assert_no_symlink_chain(Path(pins["binaryPath"]).expanduser())
        validate_runtime(output, manifest, binary)
        ledger = ledger_read(path)
        validate_ledger_profile_binding(manifest, ledger)
        counts = ledger_validate(ledger)
        if any(row.get("case") == case for row in ledger["reservedRootCalls"]):
            raise ValueError(f"root case {case!r} is already reserved; no retry or repeat")
        root_limit = policy["authorization"]["rootChatCalls"]
        if counts[0] >= root_limit:
            raise ValueError(f"{root_limit} authorized root Chat call(s) are exhausted")
        row = {
            "case": case,
            "reservedBeforeConsent": True,
            "logicalChatRequests": 1,
            "attemptState": "reserved",
            "attemptOutcome": "pending",
            "reservedAtUtc": datetime.now(timezone.utc).isoformat(),
        }
        ledger["reservedRootCalls"].append(row)
        counts = ledger_validate(ledger)
        ledger_save(path, ledger)
    print(json.dumps({"authorizationProfile": profile_name, "reservedRootChatCalls": counts[0], "rootChatLimit": root_limit, "case": case}, sort_keys=True))


def _skill_ledger_change(path: Path, case: str, operation: str, fields: dict[str, object] | None = None) -> dict[str, object]:
    if operation not in {"reserve", "consume", "settle"}:
        raise ValueError("invalid native ledger operation")
    path = assert_no_symlink_chain(Path(path).expanduser()).resolve(strict=True)
    if path.name != "ledger.json" or path.parent.name != "evidence":
        raise ValueError("native ledger transition requires the canonical prepared evidence/ledger.json")
    prepared, _, _ = native_prepared_binding(path.parent.parent, path)
    with ledger_lock(path):
        _, manifest, ledger = native_prepared_binding(prepared, path)
        validate_ledger_profile_binding(manifest, ledger)
        matches = [row for row in ledger["nativeSessions"] if row.get("case") == case]
        if operation == "reserve":
            if ledger.get("nativeAuthorization", {}).get("state") != "authorized":
                raise ValueError("native skill sessions remain blocked pending N2; no launch or reservation allowed")
            if matches:
                raise ValueError(f"native case {case!r} already has a reservation; no retry/relaunch")
            root_calls, sessions, observed_turns = ledger_validate(ledger)
            policy = authorization_profile(ledger["wave"])
            authorization = policy["authorization"]
            turn_limit = authorization["logicalTurnsPerNativeSession"]
            if sessions >= authorization["nativeSkillSessions"] or root_calls + turn_limit * (sessions + 1) > authorization["logicalRequestsPerWave"] or root_calls + observed_turns > authorization["logicalRequestsPerWave"]:
                raise ValueError("native session ceiling or actual logical-request envelope is exhausted")
            if any(isinstance(row.get("observedLogicalTurns"), int) and row["observedLogicalTurns"] > turn_limit for row in ledger["nativeSessions"]):
                raise ValueError("a prior native session exceeded its authorized turn limit; no further session")
            row = {
                "case": case,
                "reservedBeforeLaunch": True,
                "maximumLogicalTurns": turn_limit,
                "attemptState": "reserved",
                "outcome": "pending",
                "observedLogicalTurns": None,
                "retryEvents": None,
                "reservedAtUtc": datetime.now(timezone.utc).isoformat(),
            }
            ledger["nativeSessions"].append(row)
        elif operation == "consume":
            if len(matches) != 1 or matches[0].get("attemptState") != "reserved" or matches[0].get("outcome") != "pending":
                raise ValueError("native launch requires exactly one unused matching session reservation")
            row = matches[0]
            row["attemptState"] = "consumed"
            row["outcome"] = "unknown"
            row["consumedAtUtc"] = datetime.now(timezone.utc).isoformat()
        elif operation == "settle":
            row = require_native_settlement(ledger, case)
            row.update(fields or {})
            row["settledAtUtc"] = datetime.now(timezone.utc).isoformat()
        ledger_validate(ledger)
        ledger_save(path, ledger)
        return dict(row)


def command_ledger_reserve_skill(args: argparse.Namespace) -> None:
    if args.case not in SKILL_CASES or args.case == "noninteractive":
        raise ValueError("native session reservation requires one plan-bearing S10 skill case")
    path = Path(args.ledger)
    row = _skill_ledger_change(path, args.case, "reserve")
    authorization = authorization_profile(ledger_read(path)["wave"])["authorization"]
    print(json.dumps({"case": args.case, "attemptState": row["attemptState"], "maximumLogicalTurns": authorization["logicalTurnsPerNativeSession"], "nativeSessions": len(ledger_read(path)["nativeSessions"]), "wireRequestCount": "not measured"}, sort_keys=True))


def command_ledger_consume_skill(args: argparse.Namespace) -> None:
    if args.case not in SKILL_CASES or args.case == "noninteractive":
        raise ValueError("native session consumption requires one plan-bearing S10 skill case")
    row = _skill_ledger_change(Path(args.ledger), args.case, "consume")
    print(json.dumps({"case": args.case, "attemptState": row["attemptState"], "outcome": row["outcome"], "retryPermitted": False}, sort_keys=True))


def command_ledger_settle_skill(args: argparse.Namespace) -> None:
    output = Path(args.prepared).expanduser().resolve(strict=True)
    if args.case not in SKILL_CASES or args.case == "noninteractive":
        raise ValueError("native session settlement requires one plan-bearing S10 skill case")
    ledger_path = output / "evidence/ledger.json"
    output, _, ledger = native_prepared_binding(output, ledger_path)
    require_native_settlement(ledger, args.case)
    session = read_json(Path(args.session_evidence).expanduser())
    if not isinstance(session, dict) or session.get("schemaVersion") != 1 or session.get("channel") != "skill" or session.get("case") != args.case:
        raise ValueError("actual sanitized session evidence is missing its case binding")
    if session.get("claudeCodeVersion") != "2.1.286" or session.get("provider") != "OpenCode Go Anthropic Messages compatibility gateway" or session.get("model") != "deepseek-v4-flash":
        raise ValueError("failed native session route/version differs from the pinned attempt")
    if session.get("skillInvocation") != "/lds" or session.get("manualPermissionReview") is not True:
        raise ValueError("session evidence must preserve the actual /lds invocation and manual permission review")
    session_exit = session.get("sessionExitCode")
    operation_exit = session.get("operationExitCode")
    turns = session.get("logicalModelTurns")
    if not isinstance(session_exit, int) or session_exit < 0 or session_exit > 255 or (operation_exit is not None and (not isinstance(operation_exit, int) or operation_exit < 0 or operation_exit > 255)) or not isinstance(turns, int) or turns < 0:
        raise ValueError("actual native process exit/turn observations are malformed")
    retry_events = validate_retry_events(session.get("visibleRetryEvents"))
    installed_hash = session.get("installedSkillSha256")
    if not isinstance(installed_hash, str) or not re.fullmatch(r"[0-9a-f]{64}", installed_hash):
        raise ValueError("installed skill evidence lacks its nonsecret pinned content hash")
    try:
        expected_skill_hash = sha256((output / "home/.claude/skills/lds/SKILL.md").read_bytes())
    except OSError as error:
        raise ValueError("isolated installed skill bytes are unavailable for failed-session binding") from error
    if installed_hash != expected_skill_hash:
        raise ValueError("session installed-skill hash differs from the isolated fixture install")

    output, _, ledger = native_prepared_binding(output, ledger_path)
    require_native_settlement(ledger, args.case)
    _, _, prior_turns = ledger_validate(ledger)
    authorization = authorization_profile(ledger["wave"])["authorization"]
    turn_limit = authorization["logicalTurnsPerNativeSession"]
    request_limit = authorization["logicalRequestsPerWave"]
    success = (
        session_exit == 0
        and operation_exit == expected_operation_exit(args.case)
        and 1 <= turns <= turn_limit
        and session.get("outcome") == expected_skill_outcome(args.case)
    )
    violation = turns > turn_limit or len(ledger["reservedRootCalls"]) + prior_turns + turns > request_limit
    ledger_outcome = "completed" if success else "stopped" if session_exit in {130, 143} else "failed"
    summary = {
        "schemaVersion": 1,
        "channel": "skill",
        "case": args.case,
        "status": "completed" if success else ledger_outcome,
        "claudeCodeVersion": "2.1.286",
        "provider": "OpenCode Go Anthropic Messages compatibility gateway",
        "model": "deepseek-v4-flash",
        "sessionExitCode": session_exit,
        "operationExitCode": operation_exit,
        "logicalModelTurns": turns,
        "visibleRetryEvents": retry_events,
        "wireRequestCount": "not measured; not a proven cap",
        "skillInvocation": "/lds",
        "manualPermissionReview": True,
        "installedSkillSha256": installed_hash,
        "planDigest": None,
        "rawInputStored": False,
        "rawStreamsStored": False,
        "retryPermitted": False,
        "authorizationViolation": violation,
    }
    destination = output / "evidence/live/skill" / args.case / "session-observation.json"
    if destination.exists():
        raise ValueError("native session observation already exists; no retry or replacement")
    output, _, ledger = native_prepared_binding(output, ledger_path)
    require_native_settlement(ledger, args.case)
    atomic_json(destination, summary)
    row = _skill_ledger_change(ledger_path, args.case, "settle", {
        "outcome": ledger_outcome,
        "sessionExitCode": session_exit,
        "operationExitCode": operation_exit,
        "observedLogicalTurns": turns,
        "retryEvents": retry_events,
        "authorizationViolation": violation,
        "retryPermitted": False,
    })
    print(json.dumps({"case": args.case, "outcome": row["outcome"], "sessionExitCode": session_exit, "operationExitCode": operation_exit, "logicalModelTurns": turns, "visibleRetryEvents": retry_events, "wireRequestCount": "not measured", "authorizationViolation": violation, "retryPermitted": False}, sort_keys=True))


def read_until_fd(descriptor: int, marker: bytes, timeout: float = 10.0) -> bytes:
    deadline = time.monotonic() + timeout
    accumulated = bytearray()
    while marker not in accumulated:
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError(f"offline PTY test did not observe mock marker {marker!r}")
        ready, _, _ = select.select([descriptor], [], [], remaining)
        if not ready:
            continue
        try:
            chunk = os.read(descriptor, 4096)
        except OSError as error:
            raise RuntimeError("offline PTY closed before its expected mock marker") from error
        if not chunk:
            raise RuntimeError("offline PTY reached EOF before its expected mock marker")
        accumulated.extend(chunk)
    return bytes(accumulated)


def make_mock_cli(binary: Path, config_path: Path) -> None:
    source = r'''#!/usr/bin/env python3
import hashlib, json, os, sys, termios, time
from pathlib import Path

CONFIG = json.loads(Path(@CONFIG@).read_text())

def sha(data):
    return hashlib.sha256(data).hexdigest()

def plan_for(target):
    case = Path(target).name
    classifications = {
        "collision": ["create", "different-collision", "create", "create"],
        "partial-legacy": ["refusal"] * 4,
    }.get(case, ["create"] * 4)
    outputs = []
    for (path, content), classification in zip(CONFIG["outputs"].items(), classifications):
        data = content.encode()
        outputs.append({"path": path, "classification": classification, "content": content, "contentSha256": sha(data), "reason": "offline mock only", "asset": {"source": "embedded", "sha256": sha(data)}})
    findings = []
    if case == "collision":
        findings = [{"code": "refusal.output-collision", "path": "docs/reference/usage.md"}]
    elif case == "partial-legacy":
        findings = [{"code": "refusal.legacy-docs", "path": "docs/"}]
    plan = {
        "schemaVersion": 1, "operation": "docs plan", "targetRoot": str(target), "targetIdentity": "offline-mock-target",
        "factsDigest": "sha256:" + CONFIG["factsSha256"], "outputs": outputs,
        "necessaryParentDirectories": ["docs/", "docs/reference/", "engineering/", "engineering/product/"],
        "existingParentDirectories": [], "preserved": [], "reportOnly": [], "veto": case in ("collision", "partial-legacy"), "findings": findings, "skipped": [],
    }
    plan["planDigest"] = "sha256:" + sha(json.dumps(plan, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode())
    return plan

def diagnostics_for(target):
    case = Path(target).name
    before = CONFIG["beforeSnapshots"][case]
    entries = {entry["path"]: entry for entry in before["entries"]}
    files = [{"path": path, "sha256": entry["sha256"], "size": entry["size"]}
             for path, entry in sorted(entries.items())
             if entry.get("type") == "file" and path.lower().endswith(".md")
             and (path == "README.md" or path.startswith(("docs/", "engineering/")))]
    links = []
    findings = []
    if case == "collision":
        links = [{"source": "docs/reference/usage.md", "line": 5, "target": "../howto.md", "kind": "inline"}]
        findings = [{"code": "validation.local-target-missing", "path": "docs/reference/usage.md", "line": 5, "target": "../howto.md", "message": "local file target is missing: ../howto.md"}]
    return (
        {"schemaVersion": 1, "operation": "docs inventory", "files": files, "links": links, "skipped": []},
        {"schemaVersion": 1, "operation": "docs validate", "findings": findings, "skipped": []},
    )

argv = sys.argv[1:]
if argv[:2] == ["plumbing", "adopt"]:
    use_override = os.environ.get("XDG_CONFIG_HOME", "").endswith("/root-cli/override")
    asset = CONFIG["overrideFlowAsset"] if use_override else CONFIG["flowAsset"]
    content = CONFIG["overrideFlowContent"] if use_override else CONFIG["flowContent"]
    print(json.dumps({"operation": "adopt", "asset": asset, "content": content}))
elif argv[:3] == ["plumbing", "docs", "inventory"]:
    target = Path(argv[3])
    print(json.dumps(diagnostics_for(target)[0]))
elif argv[:3] == ["plumbing", "docs", "validate"]:
    target = Path(argv[3])
    print(json.dumps(diagnostics_for(target)[1]))
    if target.name == "collision":
        raise SystemExit(1)
elif argv[:3] == ["plumbing", "docs", "plan"]:
    target = Path(argv[3])
    plan = plan_for(target)
    print(json.dumps(plan))
    if plan["veto"]:
        raise SystemExit(3)
elif argv and argv[0] == "adopt":
    target = Path(argv[1])
    case = target.name
    use_override = os.environ.get("XDG_CONFIG_HOME", "").endswith("/root-cli/override")
    asset = CONFIG["overrideFlowAsset"] if use_override else CONFIG["flowAsset"]
    plan = plan_for(target)
    inventory, validation = diagnostics_for(target)
    inventory_body = {key: value for key, value in inventory.items() if key not in {"schemaVersion", "operation"}}
    validation_body = {key: value for key, value in validation.items() if key not in {"schemaVersion", "operation"}}
    if case == "noninteractive":
        os.write(2, b"refusal.interactive-required")
        raise SystemExit(3)
    if plan["veto"]:
        print(json.dumps({"schemaVersion": 1, "operation": "adopt", "targetRoot": str(target), "flowAsset": CONFIG["flowAsset"], "advisory": "not requested for veto", "status": "refused", "inventory": inventory_body, "validation": validation_body, "plan": plan, "apply": None}))
        raise SystemExit(3)
    paths = CONFIG["mockPaths"][case]
    Path(paths["pid"]).write_text(str(os.getpid()))
    Path(paths["tty"]).write_text(json.dumps({"stdin": os.ttyname(0), "stderr": os.ttyname(2), "stdinIsTty": os.isatty(0), "stderrIsTty": os.isatty(2)}))
    os.write(2, b"MOCK SEND> ")
    if sys.stdin.readline().strip() != "send":
        raise SystemExit(21)
    Path(paths["requests"]).write_text("1")
    Path(paths["pending"]).write_text("mock transport pending; no network")
    while not Path(paths["release"]).exists():
        time.sleep(0.01)
    termios.tcflush(0, termios.TCIFLUSH)
    os.write(2, ("MOCK FULL PLAN " + json.dumps(plan, sort_keys=True) + "\nMOCK APPLY> ").encode())
    if sys.stdin.readline().strip() != "apply " + plan["planDigest"]:
        raise SystemExit(22)
    result = {
        "schemaVersion": 1, "operation": "adopt", "targetRoot": str(target), "flowAsset": asset,
        "advisory": {"mockOnly": True}, "status": "mock-complete", "inventory": inventory_body, "validation": validation_body, "plan": plan, "apply": None,
    }
    mutation = CONFIG.get("workflowMutations", {}).get(case)
    if mutation == "malformed-links":
        result["inventory"]["links"] = ["not-a-link-object"]
    elif mutation == "changed-validation":
        result["validation"]["findings"] = [{"code": "validation.local-target-missing", "path": "README.md", "line": 1, "target": "missing.md", "message": "mutated mock diagnostic"}]
    elif mutation == "missing-validation":
        del result["validation"]
    print(json.dumps(result))
else:
    raise SystemExit(64)
'''.replace("@CONFIG@", repr(str(config_path)))
    binary.write_text(source)
    binary.chmod(0o755)


def offline_native_prewrite_regression(root: Path, binary: Path) -> str:
    cases_root = root / "native-prewrite-regressions"
    cases_root.mkdir()

    def pin_prepared(prepared: Path, binary_path: Path = binary) -> None:
        with redirect_stdout(io.StringIO()):
            command_pin_runtime(argparse.Namespace(prepared=str(prepared), binary=str(binary_path)))

    def clone_prepared(source: Path, name: str, binary_path: Path = binary) -> Path:
        destination = cases_root / name
        shutil.copytree(source, destination)
        manifest_path = destination / "evidence/prepared.json"
        manifest = read_json(manifest_path)
        manifest["preparedRoot"] = str(destination)
        atomic_json(manifest_path, manifest)
        (destination / "evidence/runtime-pins.json").unlink()
        pin_prepared(destination, binary_path)
        return destination

    def commands(prepared: Path) -> tuple[tuple[str, object], ...]:
        ledger_path = prepared / "evidence/ledger.json"
        return (
            ("reserve", lambda: command_ledger_reserve_skill(argparse.Namespace(ledger=str(ledger_path), case="fresh"))),
            ("consume", lambda: command_ledger_consume_skill(argparse.Namespace(ledger=str(ledger_path), case="fresh"))),
            ("settle", lambda: command_ledger_settle_skill(argparse.Namespace(prepared=str(prepared), case="fresh", session_evidence=str(cases_root / "absent-session.json")))),
        )

    def assert_refused_without_changes(prepared: Path, expected_error: str | None, label: str) -> None:
        for operation, invoke in commands(prepared):
            before = sha256(stable_json(snapshot_tree(prepared)))
            try:
                invoke()
            except (OSError, ValueError) as error:
                if expected_error is not None and expected_error not in str(error):
                    raise ValueError(f"{label} {operation} refused for an unexpected reason: {error}") from error
            else:
                raise ValueError(f"{label} {operation} unexpectedly succeeded")
            after = sha256(stable_json(snapshot_tree(prepared)))
            if after != before:
                raise ValueError(f"{label} {operation} changed prepared files before refusing")

    executor = cases_root / "executor-base"
    prepare(executor)
    pin_prepared(executor)
    executor_ledger_path = executor / "evidence/ledger.json"
    executor_ledger = ledger_read(executor_ledger_path)
    executor_ledger["nativeAuthorization"] = {"state": "authorized", "decision": "synthetic offline pre-write regression only"}
    atomic_json(executor_ledger_path, executor_ledger)

    supplemental = cases_root / "supplemental-base"
    prepare(supplemental, "supplemental-decline")
    pin_prepared(supplemental)
    assert_refused_without_changes(supplemental, "prepared authorization profile grants zero native skill sessions", "unchanged supplemental profile")
    if (supplemental / "evidence/live/skill/fresh/session-observation.json").exists():
        raise ValueError("supplemental settlement refusal created a native session observation")

    for name, pin_mutation in (("promotion-missing-pins", "missing"), ("promotion-stale-pins", "stale")):
        promoted = clone_prepared(supplemental, name)
        ledger_path = promoted / "evidence/ledger.json"
        ledger = ledger_read(ledger_path)
        ledger["wave"] = "executor"
        ledger["authorization"] = dict(EXECUTOR_AUTHORIZATION)
        ledger["nativeAuthorization"] = {"state": "authorized", "decision": "mutated synthetic test state"}
        atomic_json(ledger_path, ledger)
        pins_path = promoted / "evidence/runtime-pins.json"
        if pin_mutation == "missing":
            pins_path.unlink()
        else:
            pins = read_json(pins_path)
            pins["helperSha256"] = "0" * 64
            atomic_json(pins_path, pins)
        assert_refused_without_changes(promoted, "prepared authorization profile grants zero native skill sessions", name)

    invalid_runtime_states: list[tuple[str, str | None]] = [
        ("missing-runtime-pins", "missing or invalid executable/runtime pins"),
        ("missing-manifest", None),
        ("changed-manifest", "runtime pins do not bind to this preparation manifest"),
        ("missing-binary", None),
        ("changed-binary", "root CLI executable differs from its pre-pinned bytes"),
        ("missing-helper", None),
        ("changed-helper", "S10 helper differs from the runtime-pinned bytes"),
    ]
    original_helper = Path(__file__).resolve()
    helper_copy = cases_root / "drifted-helper.py"
    helper_copy.write_bytes(original_helper.read_bytes() + b"\n# synthetic helper drift regression\n")

    for name, expected_error in invalid_runtime_states:
        binary_path = binary
        if name in {"missing-binary", "changed-binary"}:
            binary_path = cases_root / f"{name}-binary"
            shutil.copy2(binary, binary_path)
        prepared = clone_prepared(executor, name, binary_path)
        if name == "missing-runtime-pins":
            (prepared / "evidence/runtime-pins.json").unlink()
        elif name == "missing-manifest":
            (prepared / "evidence/prepared.json").unlink()
        elif name == "changed-manifest":
            manifest_path = prepared / "evidence/prepared.json"
            manifest = read_json(manifest_path)
            manifest["testOnlyPinDrift"] = True
            atomic_json(manifest_path, manifest)
        elif name == "missing-binary":
            binary_path.unlink()
        elif name == "changed-binary":
            binary_path.write_bytes(binary_path.read_bytes() + b"\n# synthetic binary drift regression\n")
        elif name == "missing-helper":
            globals()["__file__"] = str(cases_root / "absent-helper.py")
        elif name == "changed-helper":
            globals()["__file__"] = str(helper_copy)

        try:
            assert_refused_without_changes(prepared, expected_error, name)
        finally:
            if name in {"missing-helper", "changed-helper"}:
                globals()["__file__"] = str(original_helper)

    return "PASS: reserve/consume/settle refused promoted supplemental state, zero-native settlement, missing/stale pins, and missing/changed manifest/helper/binary; complete prepared trees stayed byte/path/type/mode-identical with no lock or session-observation writes"


def offline_regressions() -> dict[str, object]:
    results: dict[str, object] = {}
    with tempfile.TemporaryDirectory(prefix="lds-bds251-s10-regressions-") as temporary:
        root = Path(temporary)
        output = root / "prepared"
        prepare(output)
        mock_dir = root / "mock"
        mock_dir.mkdir()
        binary = mock_dir / "lds-mock"
        mock_paths: dict[str, dict[str, str]] = {}
        for case in ("fresh", "partial-engineering", "decline", "stale", "override"):
            mock_paths[case] = {name: str(mock_dir / f"{case}.{name}") for name in ("pid", "tty", "pending", "release", "requests")}
        flow = (ROOT / "assets/flows/adopt.md").read_text(encoding="utf-8")
        override_flow = (output / "evidence/override-seed/adopt.md").read_text(encoding="utf-8")
        config = {
            "flowContent": flow,
            "flowAsset": {"relativeName": "flows/adopt.md", "source": "embedded", "sha256": sha256(flow.encode())},
            "overrideFlowContent": override_flow,
            "overrideFlowAsset": {"relativeName": "flows/adopt.md", "source": "override", "sha256": sha256(override_flow.encode())},
            "factsSha256": sha256((GREENFIELD / "facts.json").read_bytes()),
            "outputs": {path: (GREENFIELD / "expected" / path).read_text(encoding="utf-8") for path in APPROVED_OUTPUTS},
            "beforeSnapshots": {
                case.split("/", 1)[1]: read_json(output / manifest_relative)
                for case, manifest_relative in load_manifest(output)["beforeSnapshots"].items()
                if case.startswith("root-cli/")
            },
            "mockPaths": mock_paths,
            "workflowMutations": {},
        }
        config_path = mock_dir / "config.json"
        config_path.write_bytes(stable_json(config))
        make_mock_cli(binary, config_path)
        with redirect_stdout(io.StringIO()):
            command_pin_runtime(argparse.Namespace(prepared=str(output), binary=str(binary)))

        manifest = load_manifest(output)
        pinned_path = output / manifest["beforeSnapshots"]["root-cli/fresh"]
        pinned_bytes = pinned_path.read_bytes()
        pinned_path.write_bytes(b"{}\n")
        try:
            validate_runtime(output, manifest, binary)
        except ValueError:
            results["runtimeSnapshotTamper"] = "refused"
        else:
            raise ValueError("runtime pin regression: edited expected target snapshot was accepted")
        pinned_path.write_bytes(pinned_bytes)
        binary_bytes = binary.read_bytes()
        binary.write_bytes(binary_bytes + b"\n# changed after runtime pin\n")
        try:
            validate_runtime(output, manifest, binary)
        except ValueError:
            results["runtimeBinaryTamper"] = "refused"
        else:
            raise ValueError("runtime pin regression: changed binary was accepted")
        binary.write_bytes(binary_bytes)
        binary.chmod(0o755)
        validate_runtime(output, manifest, binary)

        supplemental_output = root / "supplemental-decline"
        with redirect_stdout(io.StringIO()):
            prepare(supplemental_output, "supplemental-decline")
            command_pin_runtime(argparse.Namespace(prepared=str(supplemental_output), binary=str(binary)))
        supplemental_manifest = load_manifest(supplemental_output)
        supplemental_ledger_path = supplemental_output / "evidence/ledger.json"
        supplemental_ledger = ledger_read(supplemental_ledger_path)
        if manifest_profile_name(supplemental_manifest) != "supplemental-decline" or supplemental_manifest["matrix"]["rootProviderCases"] != ["decline"]:
            raise ValueError("supplemental profile regression: preparation did not bind only the decline root case")
        if ledger_validate(supplemental_ledger) != (0, 0, 0) or supplemental_ledger["reservedRootCalls"] or supplemental_ledger["nativeSessions"] or supplemental_ledger["nativeAuthorization"].get("state") != "blocked":
            raise ValueError("supplemental profile regression: initial ledger is not empty with native authorization blocked")
        if supplemental_ledger["authorization"] != {
            "rootChatCalls": 1,
            "nativeSkillSessions": 0,
            "logicalTurnsPerNativeSession": 0,
            "logicalRequestsPerWave": 1,
            "wireHTTPRequests": "not a proven cap; client/gateway retries are separately reported",
        }:
            raise ValueError("supplemental profile regression: authorization is not a one-request/no-native profile")

        try:
            command_run_root(argparse.Namespace(prepared=str(supplemental_output), binary=str(binary), case="fresh", channel="root-cli"))
        except ValueError as error:
            if "outside the supplemental-decline authorization profile" not in str(error):
                raise
        else:
            raise ValueError("supplemental profile regression: run-root accepted a non-decline case")
        try:
            command_ledger_reserve_root(argparse.Namespace(ledger=str(supplemental_ledger_path), case="fresh"))
        except ValueError as error:
            if "not authorized by the supplemental-decline profile" not in str(error):
                raise
        else:
            raise ValueError("supplemental profile regression: reservation accepted a second root case")

        for changed_authorization in (
            {**supplemental_ledger["authorization"], "rootChatCalls": 2},
            {**supplemental_ledger["authorization"], "nativeSkillSessions": 1},
            {**supplemental_ledger["authorization"], "logicalRequestsPerWave": 2},
            {**supplemental_ledger["authorization"], "rootChatCalls": True},
        ):
            changed = json.loads(stable_json(supplemental_ledger))
            changed["authorization"] = changed_authorization
            try:
                ledger_validate(changed)
            except ValueError:
                continue
            raise ValueError("supplemental profile regression: mutated authorization limits were accepted")
        promoted = json.loads(stable_json(supplemental_ledger))
        promoted["wave"] = "executor"
        promoted["authorization"] = dict(EXECUTOR_AUTHORIZATION)
        try:
            validate_ledger_profile_binding(supplemental_manifest, promoted)
        except ValueError:
            pass
        else:
            raise ValueError("supplemental profile regression: executor-limit promotion escaped the pinned supplemental profile")
        native_enabled = json.loads(stable_json(supplemental_ledger))
        native_enabled["nativeAuthorization"] = {"state": "authorized", "decision": "mutated"}
        try:
            ledger_validate(native_enabled)
        except ValueError:
            pass
        else:
            raise ValueError("supplemental profile regression: native authorization mutation was accepted")
        with redirect_stdout(io.StringIO()):
            try:
                command_ledger_reserve_skill(argparse.Namespace(ledger=str(supplemental_ledger_path), case="fresh"))
            except ValueError:
                pass
            else:
                raise ValueError("supplemental profile regression: native reservation was accepted")
        results["supplementalDeclineProfile"] = "PASS: decline-only; exact one-request limit; other case, mutated limits/profile, native authorization/reservation, and executor-limit promotion refused"
        results["nativePrewriteBinding"] = offline_native_prewrite_regression(root, binary)

        with redirect_stdout(io.StringIO()):
            command_ledger_reserve_root(argparse.Namespace(ledger=str(supplemental_ledger_path), case="decline"))
        try:
            with redirect_stdout(io.StringIO()):
                command_ledger_reserve_root(argparse.Namespace(ledger=str(supplemental_ledger_path), case="decline"))
        except ValueError as error:
            if "already reserved" not in str(error):
                raise
        else:
            raise ValueError("supplemental profile regression: second decline reservation was accepted")
        if ledger_validate(ledger_read(supplemental_ledger_path))[0] != 1:
            raise ValueError("supplemental profile regression: unique decline reservation did not consume the only allocation")
        _root_ledger_change(supplemental_ledger_path, "decline", "consume", {"logicalChatRequests": 1})
        try:
            require_root_reservation(ledger_read(supplemental_ledger_path), "decline")
        except ValueError:
            pass
        else:
            raise ValueError("supplemental profile regression: consumed decline reservation allowed relaunch")
        results["supplementalDeclineNoRetry"] = "PASS: one reserved/consumed application slot; duplicate reservation and consumed relaunch refused in an isolated temporary test ledger; no provider request"

        override_bytes = (output / "evidence/override-seed/adopt.md").read_bytes()
        override = output / "xdg/skill/override/lds/flows/adopt.md"
        override.parent.mkdir(parents=True, exist_ok=True)
        override.write_bytes(override_bytes)
        root_override = output / "xdg/root-cli/override/lds/flows/adopt.md"
        root_override.parent.mkdir(parents=True, exist_ok=True)
        root_override.write_bytes(override_bytes)
        validate_case_xdg_matrix(output, manifest)
        wrong_case = output / "xdg/skill/fresh/lds/flows/adopt.md"
        wrong_case.parent.mkdir(parents=True)
        wrong_case.write_bytes(override_bytes)
        try:
            validate_case_xdg_matrix(output, manifest)
        except ValueError:
            results["caseLocalOverrideIsolation"] = "override-only accepted; cross-case override refused"
        else:
            raise ValueError("XDG isolation regression: override leaked into a baseline case")
        shutil.rmtree(wrong_case.parents[1])

        semantic_left = {
            "targetRoot": "/left", "targetIdentity": "left", "planDigest": "left-digest",
            "existingParentDirectories": [{"path": "docs", "identity": "left-parent", "mode": 493}],
            "outputs": [{"path": "docs/a", "identity": "left-output", "asset": {"identity": "left-asset"}}],
        }
        semantic_right = {
            "targetRoot": "/right", "targetIdentity": "right", "planDigest": "right-digest",
            "existingParentDirectories": [{"path": "docs", "identity": "right-parent", "mode": 493}],
            "outputs": [{"path": "docs/a", "identity": "right-output", "asset": {"identity": "right-asset"}}],
        }
        base_left = dict(semantic_left)
        base_right = dict(semantic_right)
        base_left["outputs"] = [{"path": "docs/a", "asset": {"identity": "same"}}]
        base_right["outputs"] = [{"path": "docs/a", "asset": {"identity": "same"}}]
        if semantic_plan(base_left) != semantic_plan(base_right):
            raise ValueError("semantic normalization regression: exact permitted target identities were not excluded")
        base_right["outputs"] = [{"path": "docs/a", "asset": {"identity": "different"}}]
        if semantic_plan(base_left) == semantic_plan(base_right):
            raise ValueError("semantic normalization regression: arbitrary nested identity was discarded")
        results["semanticIdentityScope"] = "only exact plan identity fields excluded"

        digest = "sha256:" + "a" * 64
        skill_target = target_for(output, manifest, "skill", "fresh")
        skill_before = read_json(output / manifest["beforeSnapshots"]["skill/fresh"])
        skill_plan, skill_digest = capture_plan(
            binary,
            skill_target,
            GREENFIELD / "facts.json",
            safe_environment("OFFLINE_DUMMY_ONLY", case_xdg(output, "skill", "fresh"), output / "home"),
            "fresh",
        )
        inventory_files = [
            {"path": path, "sha256": entry["sha256"], "size": entry["size"]}
            for path, entry in sorted(snapshot_index(skill_before).items())
            if entry.get("type") == "file"
            and path.lower().endswith(".md")
            and (path == "README.md" or path.startswith(("docs/", "engineering/")))
        ]
        tool_outputs: dict[str, object] = {
            "inventory.json": {"schemaVersion": 1, "operation": "docs inventory", "files": inventory_files, "links": [], "skipped": []},
            "validate.json": {"schemaVersion": 1, "operation": "docs validate", "findings": [], "skipped": []},
            "render.json": {
                "schemaVersion": 1,
                "operation": "docs render",
                "outputs": [
                    {
                        "path": path,
                        "content": (GREENFIELD / "expected" / path).read_text(encoding="utf-8"),
                        "sha256": OUTPUT_HASHES[path],
                        "asset": {"relativeName": path, "source": "embedded", "sha256": "a" * 64},
                    }
                    for path in APPROVED_OUTPUTS
                ],
            },
            "apply.json": {
                "schemaVersion": 1, "operation": "docs apply", "status": "applied", "planDigest": skill_digest,
                "created": [{"path": path, "sha256": OUTPUT_HASHES[path]} for path in APPROVED_OUTPUTS],
                "identicalUntouched": [], "unattempted": [], "failed": None, "preserved": [],
            },
        }
        validate_skill_tool_results(tool_outputs, skill_target, skill_before, "fresh", skill_digest)
        bad_inventory = dict(tool_outputs)
        bad_inventory["inventory.json"] = {**tool_outputs["inventory.json"], "files": []}
        for invalid_results in (bad_inventory, {key: value for key, value in tool_outputs.items() if key != "apply.json"}):
            try:
                validate_skill_tool_results(invalid_results, skill_target, skill_before, "fresh", skill_digest)
            except ValueError:
                continue
            raise ValueError("skill tool-result regression: target-mismatched inventory or absent apply artifact was accepted")
        results["skillToolResults"] = "synthetic schema mutation controls supplement the separate actual-baseline tool-artifact collector regression"

        apply_result = {
            "schemaVersion": 1, "operation": "docs apply", "status": "applied", "planDigest": digest,
            "created": [{"path": path, "sha256": OUTPUT_HASHES[path]} for path in APPROVED_OUTPUTS],
            "identicalUntouched": [], "unattempted": [], "failed": None, "preserved": [],
        }
        if safe_apply_result(apply_result) != apply_result:
            raise ValueError("docs apply result regression: valid failed:null schema was not retained")
        identity_variant = dict(apply_result)
        identity_variant["created"] = [{**entry, "identity": "different"} for entry in apply_result["created"]]
        if semantic_apply(apply_result, digest) == semantic_apply(identity_variant, digest):
            raise ValueError("semantic apply regression: nested identity-like result fields were discarded")
        for broken in ({key: value for key, value in apply_result.items() if key != "failed"}, {**apply_result, "failed": []}):
            try:
                safe_apply_result(broken)
            except ValueError:
                continue
            raise ValueError("docs apply regression: missing/null failure field or wrong failed type was accepted")
        results["applyResultShape"] = "actual failed:null schema retained; missing/wrong failure and nested-identity loss refused"

        for apply in (None, {"status": "applied", "planDigest": digest, "created": list(APPROVED_OUTPUTS)}):
            try:
                with tempfile.TemporaryDirectory(dir=root) as empty:
                    target = Path(empty)
                    for path in APPROVED_OUTPUTS:
                        (target / path).parent.mkdir(parents=True, exist_ok=True)
                    verify_created_outputs(apply, digest, snapshot_tree(target))
            except ValueError:
                continue
            raise ValueError("output verification regression: apply:null or directories-only was accepted as fresh PASS")
        results["actualOutputBytes"] = "apply:null and directories-only both refused"

        failed_session = {
            "schemaVersion": 1, "channel": "skill", "case": "fresh", "claudeCodeVersion": "2.1.286",
            "provider": "OpenCode Go Anthropic Messages compatibility gateway", "model": "deepseek-v4-flash",
            "sessionExitCode": 9, "operationExitCode": 7, "logicalModelTurns": 99,
            "toolExitCodes": {}, "skillInvocation": "/lds", "manualPermissionReview": True,
            "installedSkillSha256": "a" * 64, "visibleRetryEvents": "visible-retries:2", "outcome": "failed",
        }
        try:
            validate_skill_session("fresh", failed_session, 7, 9, 99, "visible-retries:2", "failed")
        except ValueError:
            results["skillFailureCannotPass"] = "operation/session exits and 99 turns refused by PASS verifier"
        else:
            raise ValueError("skill verifier regression: nonzero exits/99 turns passed")
        tool_exit_cases = {
            "fresh": {"adopt": 0, "inventory": 0, "validate": 0, "render": 0, "plan": 0, "apply": 0},
            "collision": {"adopt": 0, "inventory": 0, "validate": 1, "render": 0, "plan": 3},
            "decline": {"adopt": 0, "inventory": 0, "validate": 0, "render": 0, "plan": 0},
            "stale": {"adopt": 0, "inventory": 0, "validate": 0, "render": 0, "plan": 0, "apply": 3},
        }
        for case, exits in tool_exit_cases.items():
            validate_skill_tool_exits(case, exits)
        for case, invalid in (
            ("fresh", {**tool_exit_cases["fresh"], "plan": 3}),
            ("stale", {**tool_exit_cases["stale"], "apply": 0}),
            ("collision", {**tool_exit_cases["collision"], "apply": 3}),
            ("collision", {**tool_exit_cases["collision"], "validate": True}),
        ):
            try:
                validate_skill_tool_exits(case, invalid)
            except ValueError:
                continue
            raise ValueError(f"skill process-exit regression: incorrect {case} tool exits were accepted")
        results["skillToolProcessExits"] = "exact reader/inventory/validate/render/plan/apply exit sets checked per case"

        collision_inventory = {
            "schemaVersion": 1,
            "operation": "docs inventory",
            "files": [
                {"path": path, "sha256": entry["sha256"], "size": entry["size"]}
                for path, entry in sorted(snapshot_index(read_json(output / manifest["beforeSnapshots"]["root-cli/collision"])).items())
                if entry.get("type") == "file" and path.lower().endswith(".md") and (path == "README.md" or path.startswith(("docs/", "engineering/")))
            ],
            "links": [{"source": "docs/reference/usage.md", "line": 5, "target": "../howto.md", "kind": "inline"}],
            "skipped": [],
        }
        collision_validation = {
            "schemaVersion": 1,
            "operation": "docs validate",
            "findings": [{"code": "validation.local-target-missing", "path": "docs/reference/usage.md", "line": 5, "target": "../howto.md", "message": "local file target is missing: ../howto.md"}],
            "skipped": [],
        }
        flow_asset = {"relativeName": "flows/adopt.md", "source": "embedded", "sha256": sha256(flow.encode())}
        diagnostic_exits = {"inventory": 0, "validate": 1}
        validate_tool_diagnostics(collision_inventory, collision_validation, diagnostic_exits, read_json(output / manifest["beforeSnapshots"]["root-cli/collision"]), "collision")
        collision_inventory_body = standalone_tool_body(collision_inventory, "docs inventory", {"files", "links", "skipped"})
        collision_validation_body = standalone_tool_body(collision_validation, "docs validate", {"findings", "skipped"})
        for invalid_inventory, invalid_validation in (
            ({**collision_inventory, "links": [{**collision_inventory["links"][0], "target": "../different.md"}]}, collision_validation),
            (collision_inventory, {**collision_validation, "findings": []}),
            ({**collision_inventory, "links": ["not-a-link-object"]}, collision_validation),
        ):
            try:
                validate_tool_diagnostics(invalid_inventory, invalid_validation, diagnostic_exits, read_json(output / manifest["beforeSnapshots"]["root-cli/collision"]), "collision")
            except ValueError:
                continue
            raise ValueError("collision diagnostic regression: a changed or malformed supported-schema artifact was accepted")
        root_result = {
            "planDigest": digest, "status": "refused", "exitCode": 3, "flowAsset": flow_asset,
            "inventory": collision_inventory_body, "validation": collision_validation_body,
            "diagnosticExits": diagnostic_exits, "apply": None,
        }
        skill_result = {
            "planDigest": digest, "outcome": "refused", "operationExitCode": 3, "sessionExitCode": 0,
            "logicalModelTurns": 1, "flowAsset": flow_asset, "inventory": collision_inventory,
            "validation": collision_validation, "diagnosticExits": diagnostic_exits, "apply": None,
        }
        try:
            semantic_result(skill_result, digest, "skill", "fresh")
        except ValueError:
            results["semanticExitScope"] = "wrong case operation exits refused; native session exit must be zero"
        else:
            raise ValueError("semantic result regression: an applied case accepted collision/refusal semantics")
        root_semantics = semantic_result(root_result, digest, "root-cli", "collision")
        skill_semantics = semantic_result(skill_result, digest, "skill", "collision")
        if {key: value for key, value in root_semantics.items() if key != "sessionExitCode"} != {key: value for key, value in skill_semantics.items() if key != "sessionExitCode"}:
            raise ValueError("semantic diagnostic regression: supported-schema collision evidence did not compare equally")
        changed_skill = dict(skill_result)
        changed_skill["validation"] = {**collision_validation, "findings": []}
        if semantic_result(root_result, digest, "root-cli", "collision") == semantic_result(changed_skill, digest, "skill", "collision"):
            raise ValueError("semantic diagnostic regression: changed validation findings were normalized away")
        for invalid_skill_result in (
            {**skill_result, "sessionExitCode": False},
            {**skill_result, "logicalModelTurns": True},
        ):
            try:
                semantic_result(invalid_skill_result, digest, "skill", "collision")
            except ValueError:
                continue
            raise ValueError("semantic result regression: boolean process/turn evidence was accepted as an integer")
        try:
            semantic_result({**root_result, "exitCode": 0}, digest, "root-cli", "collision")
        except ValueError:
            results["semanticExitScope"] = "wrong operation exits and changed diagnostics refused; applicable native session exit checked"
        else:
            raise ValueError("semantic result regression: wrong root refusal exit compared as success")

        outside = root / "outside"
        outside.mkdir()
        (outside / "guard.txt").write_text("untouched")
        target = root / "stale-target"
        target.mkdir()
        expected = snapshot_tree(target)
        original_open = open_target_root

        def replace_docs_with_symlink(path: Path) -> int:
            (target / "docs").symlink_to(outside, target_is_directory=True)
            return original_open(path)

        globals()["open_target_root"] = replace_docs_with_symlink
        try:
            try:
                insert_collision_no_follow(target, expected)
            except (OSError, ValueError):
                pass
            else:
                raise ValueError("stale controller regression: symlink ancestor race was accepted")
        finally:
            globals()["open_target_root"] = original_open
        if (outside / "docs/reference/usage.md").exists() or (outside / "guard.txt").read_text() != "untouched":
            raise ValueError("stale controller regression: symlink replacement wrote outside the target")

        rename_target = root / "stale-rename-target"
        rename_target.mkdir()
        rename_before = snapshot_tree(rename_target)
        rename_sink = root / "stale-rename-sink"
        rename_sink.mkdir()
        original_os_open = os.open
        rename_refused: list[bool] = []

        def rename_reference_at_usage(path: str | bytes, *args: object, **kwargs: object) -> int:
            if path == "usage.md":
                try:
                    os.rename(rename_target / "docs/reference", rename_sink / "reference")
                except PermissionError:
                    rename_refused.append(True)
                else:
                    raise ValueError("sealed fixture parent unexpectedly allowed reference directory rename")
            return original_os_open(path, *args, **kwargs)

        os.open = rename_reference_at_usage
        try:
            inserted_hash = insert_collision_no_follow(rename_target, rename_before)
        finally:
            os.open = original_os_open
        rename_after = snapshot_tree(rename_target)
        if rename_refused != [True] or (rename_sink / "reference").exists() or verify_snapshot_delta(
            rename_before, rename_after, {"docs": "directory", "docs/reference": "directory", "docs/reference/usage.md": "file"}
        ):
            raise ValueError("stale controller did not block the exact parent-directory rename before writing")
        if snapshot_index(rename_after)["docs/reference/usage.md"].get("sha256") != inserted_hash:
            raise ValueError("stale controller did not preserve the pinned collision after the blocked rename")

        controller_output = root / "controller-failure-prepared"
        prepare(controller_output)
        controller_manifest = load_manifest(controller_output)
        controller_target = target_for(controller_output, controller_manifest, "root-cli", "stale")
        controller_before = expected_case_before(controller_output, controller_manifest, "root-cli", "stale")
        controller_env = safe_environment("OFFLINE_DUMMY_ONLY", case_xdg(controller_output, "root-cli", "stale"), controller_output / "home")
        controller_plan, controller_digest = capture_plan(binary, controller_target, GREENFIELD / "facts.json", controller_env, "stale")
        controller_plan_path = controller_output / "evidence/live/root-cli/stale/prelaunch-plan.json"
        atomic_json(controller_plan_path, controller_plan)
        original_open = os.open
        original_fchmod = os.fchmod
        mode_history: dict[tuple[int, int], dict[str, object]] = {}

        def fail_fixture_create(path: str | bytes, *args: object, **kwargs: object) -> int:
            if path == "usage.md":
                raise OSError("offline injected create failure")
            return original_open(path, *args, **kwargs)

        def track_mode_restoration(descriptor: int, mode: int) -> None:
            details = os.fstat(descriptor)
            if not stat.S_ISDIR(details.st_mode):
                original_fchmod(descriptor, mode)
                return
            identity = (details.st_dev, details.st_ino)
            entry = mode_history.setdefault(identity, {"initial": stat.S_IMODE(details.st_mode), "requested": []})
            entry["requested"].append(mode)
            original_fchmod(descriptor, mode)

        os.open = fail_fixture_create
        os.fchmod = track_mode_restoration
        try:
            try:
                command_stale_insert(argparse.Namespace(
                    prepared=str(controller_output), channel="root-cli", target=str(controller_target), plan=str(controller_plan_path),
                ))
            except ValueError as error:
                if "STOP WITHOUT APPROVAL" not in str(error):
                    raise
            else:
                raise ValueError("stale controller create failure did not stop without approval")
        finally:
            os.open = original_open
            os.fchmod = original_fchmod
        controller_after_path = controller_output / "evidence/snapshots/stale-after-controller-root-cli.json"
        controller_event = read_json(controller_output / "evidence/events/stale-controller-root-cli.json")
        controller_after = read_json(controller_after_path)
        if snapshot_tree(controller_target) != controller_after or verify_snapshot_delta(controller_before, controller_after, {"docs": "directory", "docs/reference": "directory"}):
            raise ValueError("stale controller create failure did not retain its complete actual after-state")
        if not isinstance(controller_event, dict) or controller_event.get("controllerStatus") != "failed" or controller_event.get("approvalAllowed") is not False or controller_event.get("reviewedPlanDigest") != controller_digest or controller_event.get("beforeSnapshot") != str(Path(controller_manifest["beforeSnapshots"]["root-cli/stale"])) or controller_event.get("afterSnapshot") != str(controller_after_path.relative_to(controller_output)):
            raise ValueError("stale controller create failure did not retain a bound STOP WITHOUT APPROVAL event")
        if len(mode_history) != 2 or any(len(entry["requested"]) != 2 or entry["requested"][-1] != entry["initial"] for entry in mode_history.values()):
            raise ValueError("stale controller create failure did not restore both sealed directory modes")
        results["staleControllerContainment"] = "symlink and directory-rename adversaries wrote nothing off-target; injected create failure restored both modes and retained a whole after snapshot with approval disallowed; trusted quiescent fixture only"

        target.mkdir(exist_ok=True)
        (target / "docs").unlink(missing_ok=True)
        link_plan = parser().parse_args(["run-root", "--prepared", str(output), "--binary", str(binary), "--case", "rerun"])
        if link_plan.case != "rerun":
            raise ValueError("root CLI parser regression: rerun is not accepted")

        no_tty_env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "OPENCODE_API_KEY": "OFFLINE_DUMMY_ONLY"}
        no_tty = subprocess.run(
            [sys.executable, str(Path(__file__).resolve()), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", "fresh"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=no_tty_env, check=False,
        )
        if no_tty.returncode != 1 or b"real terminal" not in no_tty.stderr or ledger_read(output / "evidence/ledger.json")["reservedRootCalls"]:
            raise ValueError("direct terminal regression: non-TTY caller was not refused before reservation/launch")

        with tempfile.TemporaryDirectory(dir=root) as directories_only:
            target_dir = Path(directories_only)
            for path in APPROVED_OUTPUTS:
                (target_dir / path).parent.mkdir(parents=True, exist_ok=True)
            after_dirs = snapshot_tree(target_dir)
            apply_dirs = {"status": "applied", "planDigest": digest, "created": [{"path": path} for path in APPROVED_OUTPUTS]}
            try:
                verify_created_outputs(apply_dirs, digest, after_dirs)
            except ValueError:
                results["directoryOnlyOutputs"] = "refused"
            else:
                raise ValueError("directories-only regression was accepted")

        results["directTerminalControls"] = "non-TTY refused without reservation; interactive mock scheduled below"
        mock_code = binary.read_text()
        if any(pattern in mock_code for pattern in ("import urllib", "from urllib", "import requests", "import http.client", "from http.client")):
            raise ValueError("offline mock unexpectedly contains a network client")
        env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "OPENCODE_API_KEY": "OFFLINE_DUMMY_ONLY"}

        def launch(case: str) -> tuple[subprocess.Popen[bytes], int, bytes]:
            with redirect_stdout(io.StringIO()):
                command_ledger_reserve_root(argparse.Namespace(ledger=str(output / "evidence/ledger.json"), case=case))
            master_fd, slave_fd = os.openpty()
            command = [sys.executable, str(Path(__file__).resolve()), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", case]
            process = subprocess.Popen(command, stdin=slave_fd, stdout=slave_fd, stderr=slave_fd, env=env, cwd=ROOT, start_new_session=True)
            os.close(slave_fd)
            return process, master_fd, read_until_fd(master_fd, b"MOCK SEND> ")

        process, master_fd, initial = launch("fresh")
        try:
            os.write(master_fd, b"send\n")
            deadline = time.monotonic() + 10
            pending = Path(mock_paths["fresh"]["pending"])
            while not pending.exists() and time.monotonic() < deadline:
                time.sleep(0.01)
            if not pending.exists():
                raise TimeoutError("offline scheduling mock never reached its pending-response barrier")
            plan, expected_digest = capture_plan(binary, target_for(output, manifest, "root-cli", "fresh"), GREENFIELD / "facts.json", safe_environment("OFFLINE_DUMMY_ONLY", case_xdg(output, "root-cli", "fresh"), output / "home"), "fresh")
            if plan["planDigest"] != expected_digest:
                raise ValueError("offline scheduling mock plan digest changed between request and test")
            os.write(master_fd, f"apply {expected_digest}\n".encode())
            time.sleep(0.05)
            Path(mock_paths["fresh"]["release"]).write_text("release mock only")
            prompt_output = read_until_fd(master_fd, b"MOCK APPLY> ")
            os.write(master_fd, f"apply {expected_digest}\n".encode())
            process.wait(timeout=15)
            remaining = bytearray()
            while True:
                ready, _, _ = select.select([master_fd], [], [], 0.05)
                if not ready:
                    break
                try:
                    remaining.extend(os.read(master_fd, 4096))
                except OSError:
                    break
            terminal_output = initial + prompt_output + bytes(remaining)
            if process.returncode != 0:
                raise ValueError("direct-terminal mock CLI did not complete after fresh post-plan input")
            tty = read_json(Path(mock_paths["fresh"]["tty"]))
            if not tty["stdinIsTty"] or not tty["stderrIsTty"] or tty["stdin"] != tty["stderr"]:
                raise ValueError("mock CLI stdin/stderr were not attached directly to one caller terminal")
            if Path(mock_paths["fresh"]["requests"]).read_text() != "1" or b"MOCK FULL PLAN" not in terminal_output:
                raise ValueError("scheduling adversary did not pause after exactly one mock request and display the full plan")
            result = read_json(output / "evidence/live/root-cli/fresh/result.json")
            attempt = read_json(output / "evidence/live/root-cli/fresh/attempt.json")
            if result.get("actualWorkflowDiagnostics") != "matched-independent-prelaunch-probes" or result.get("rawInputStored") is not False or result.get("rawStreamsStored") is not False or attempt.get("humanInputRelayed") is not False or attempt.get("retryPermitted") is not False:
                raise ValueError("direct-terminal run lost matched actual diagnostics or persisted/relayed human input")
            if any("raw" in item.name for item in (output / "evidence/live/root-cli/fresh").iterdir()):
                raise ValueError("direct-terminal harness left a raw file")
            results["queuedInputAdversary"] = "PASS with fake transport only: early digest flushed by child; only fresh post-plan mock input accepted"
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            os.close(master_fd)

        process, master_fd, initial = launch("partial-engineering")
        try:
            os.write(master_fd, b"send\n")
            deadline = time.monotonic() + 10
            pending = Path(mock_paths["partial-engineering"]["pending"])
            while not pending.exists() and time.monotonic() < deadline:
                time.sleep(0.01)
            if not pending.exists():
                raise TimeoutError("interrupt mock never reached its pending-response barrier")
            child_pid = int(Path(mock_paths["partial-engineering"]["pid"]).read_text())
            process.send_signal(signal.SIGTERM)
            process.wait(timeout=15)
            if process.returncode != 130:
                diagnostic = bytearray(initial)
                while True:
                    ready, _, _ = select.select([master_fd], [], [], 0.05)
                    if not ready:
                        break
                    try:
                        diagnostic.extend(os.read(master_fd, 4096))
                    except OSError:
                        break
                raise ValueError(f"SIGTERM cleanup regression returned {process.returncode}, expected 130; mock PTY tail={bytes(diagnostic[-1200:])!r}")
            result = read_json(output / "evidence/live/root-cli/partial-engineering/result.json")
            attempt = read_json(output / "evidence/live/root-cli/partial-engineering/attempt.json")
            ledger = ledger_read(output / "evidence/ledger.json")
            row = next(item for item in ledger["reservedRootCalls"] if item["case"] == "partial-engineering")
            if result.get("status") != "stopped-unknown" or attempt.get("state") != "stopped-unknown" or result.get("afterSnapshot") is None or row.get("attemptOutcome") != "stopped-unknown" or row.get("retryPermitted") is not False:
                raise ValueError("SIGTERM cleanup regression did not retain sanitized unknown outcome, after snapshot, and consumed count")
            try:
                os.kill(child_pid, 0)
            except ProcessLookupError:
                pass
            else:
                raise ValueError("SIGTERM cleanup regression left its mock CLI child alive")
            if any("raw" in path.name for path in (output / "evidence/live/root-cli/partial-engineering").iterdir()):
                raise ValueError("SIGTERM cleanup regression left raw files")
            results["interruptCleanup"] = "PASS: mock child stopped, after snapshot and stopped-unknown ledger retained, no raw files/retry"
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            os.close(master_fd)

        helper_path = Path(__file__).resolve()

        def run_veto_control(case: str) -> None:
            master_fd, slave_fd = os.openpty()
            process = subprocess.Popen(
                [sys.executable, str(helper_path), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", case],
                stdin=slave_fd, stdout=slave_fd, stderr=slave_fd, env=env, cwd=ROOT,
            )
            os.close(slave_fd)
            try:
                process.wait(timeout=15)
                if process.returncode != 0:
                    diagnostic = bytearray()
                    while True:
                        ready, _, _ = select.select([master_fd], [], [], 0.05)
                        if not ready:
                            break
                        try:
                            diagnostic.extend(os.read(master_fd, 4096))
                        except OSError:
                            break
                    raise ValueError(f"offline full-controls pack regression {case} root helper exited {process.returncode}: {bytes(diagnostic[-1200:])!r}")
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
                os.close(master_fd)

        for veto_case in ("collision", "partial-legacy"):
            run_veto_control(veto_case)
            veto_plan = read_json(output / "evidence/live/root-cli" / veto_case / "reviewed-plan.json")
            _, veto_digest = verify_plan_shape(veto_plan, target_for(output, manifest, "root-cli", veto_case), veto_case)
            checkpoint_path = output / "evidence/human-checkpoints/root-cli" / f"{veto_case}.json"
            atomic_json(checkpoint_path, {
                "schemaVersion": 1, "testOnly": True, "synthetic": True, "ownerAuthored": True,
                "channel": "root-cli", "case": veto_case, "providerConsent": "not-requested",
                "reviewedPlanDigest": veto_digest, "fullPlanPersonallyReviewed": True,
                "decision": "veto-no-confirmation", "decisionPersonallyTyped": False,
                "typedDecision": "not-applicable",
            })
            with redirect_stdout(io.StringIO()):
                command_verify_outcome(argparse.Namespace(
                    prepared=str(output), case=veto_case,
                    result=f"evidence/live/root-cli/{veto_case}/result.json",
                    before=f"evidence/snapshots/run-before/root-cli-{veto_case}.json",
                    after=f"evidence/snapshots/after/root-cli-{veto_case}.json",
                ))

        noninteractive = subprocess.run(
            [sys.executable, str(helper_path), "run-root", "--prepared", str(output), "--binary", str(binary), "--case", "noninteractive"],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env, cwd=ROOT, check=False,
        )
        if noninteractive.returncode != 0:
            raise ValueError("offline full-controls pack regression noninteractive root helper did not settle")
        with redirect_stdout(io.StringIO()):
            command_verify_control(argparse.Namespace(prepared=str(output), channel="root-cli"))
        packaged_controls = package_root_control_cases(output, manifest, offline_only=False)
        if [item["case"] for item in packaged_controls] != ["collision", "partial-legacy", "noninteractive"]:
            raise ValueError("offline full-controls pack regression did not package both vetoes and the no-plan control")
        if packaged_controls[-1]["result"].get("status") is not None or packaged_controls[-1]["verified"].get("plan") != "none-by-contract":
            raise ValueError("offline full-controls pack regression relabeled the noninteractive no-plan result")
        results["fullControlsPack"] = "PASS: both synthetic veto controls plus actual-shape noninteractive status:null/no-plan verify-control PASS packaged; checkpoints test-only, not HUMAN-A evidence"

        mutation_cases = (
            ("decline", "malformed-links"),
            ("stale", "changed-validation"),
            ("override", "missing-validation"),
        )
        rejected_mutations: list[str] = []
        for case, mutation in mutation_cases:
            config["workflowMutations"] = {case: mutation}
            config_path.write_bytes(stable_json(config))
            process, master_fd, initial = launch(case)
            try:
                os.write(master_fd, b"send\n")
                pending = Path(mock_paths[case]["pending"])
                deadline = time.monotonic() + 10
                while not pending.exists() and time.monotonic() < deadline:
                    time.sleep(0.01)
                if not pending.exists():
                    raise TimeoutError(f"actual-result mutation mock {case} did not reach its pending barrier")
                case_target = target_for(output, manifest, "root-cli", case)
                case_before = read_json(output / manifest["beforeSnapshots"][f"root-cli/{case}"])
                case_environment = safe_environment("OFFLINE_DUMMY_ONLY", case_xdg(output, "root-cli", case), output / "home")
                case_plan, case_digest = capture_plan(binary, case_target, GREENFIELD / "facts.json", case_environment, case)
                if case_plan["planDigest"] != case_digest:
                    raise ValueError("actual-result mutation mock plan digest changed before release")
                os.write(master_fd, f"apply {case_digest}\n".encode())
                time.sleep(0.05)
                Path(mock_paths[case]["release"]).write_text("release mock only")
                prompt_output = read_until_fd(master_fd, b"MOCK APPLY> ")
                os.write(master_fd, f"apply {case_digest}\n".encode())
                process.wait(timeout=15)
                if process.returncode != 1:
                    raise ValueError(f"actual-result mutation {mutation} returned {process.returncode}, expected a retained failure")
                result_path = output / "evidence/live/root-cli" / case / "result.json"
                result = read_json(result_path)
                attempt = read_json(result_path.parent / "attempt.json")
                ledger = ledger_read(output / "evidence/ledger.json")
                row = next(item for item in ledger["reservedRootCalls"] if item["case"] == case)
                if result.get("status") != "failed-or-unknown" or result.get("actualWorkflowDiagnostics") != "actual-result-diagnostics-rejected" or attempt.get("actualWorkflowDiagnostics") != "actual-result-diagnostics-rejected" or row.get("attemptOutcome") != "failed-or-unknown" or row.get("retryPermitted") is not False:
                    raise ValueError(f"actual-result mutation {mutation} was laundered or not retained as a failed consumed attempt")
                if result.get("rawInputStored") is not False or result.get("rawStreamsStored") is not False or snapshot_tree(case_target) != case_before or result_path.parent.joinpath("verified.json").exists():
                    raise ValueError(f"actual-result mutation {mutation} retained raw data, changed the target, or emitted PASS")
                rejected_mutations.append(mutation)
            finally:
                if process.poll() is None:
                    process.send_signal(signal.SIGTERM)
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()
                os.close(master_fd)
        config["workflowMutations"] = {}
        config_path.write_bytes(stable_json(config))
        if rejected_mutations != [mutation for _, mutation in mutation_cases]:
            raise ValueError("actual adopt workflow output mutations were not all rejected")
        results["actualWorkflowDiagnosticMutations"] = "PASS: malformed inventory, changed validation, and missing validation in actual mock-transport adopt results all settled failed/unknown with after snapshots, consumed no-retry ledger rows, no raw capture, and no PASS"

        ledger = ledger_read(output / "evidence/ledger.json")
        if len(ledger["reservedRootCalls"]) != 5:
            raise ValueError("disposable mock request ledger did not retain all five synthetic application attempts")
        results["ledger"] = "five mock application attempts are isolated in this disposable synthetic ledger; each slot consumed once with no retry; providerRequests=0"

        boundary_cases = sorted(ROOT_CALL_CASES)
        boundary_ledger = ledger_read(output / "evidence/ledger.json")
        boundary_ledger["reservedRootCalls"] = [
            {"case": case, "logicalChatRequests": 1, "attemptState": "reserved", "attemptOutcome": "pending"}
            for case in boundary_cases[:5]
        ]
        if require_root_reservation(boundary_ledger, boundary_cases[4])["case"] != boundary_cases[4]:
            raise ValueError("five-call boundary did not accept its matching reserved case")
        boundary_ledger["reservedRootCalls"].append({
            "case": boundary_cases[5], "logicalChatRequests": 1, "attemptState": "reserved", "attemptOutcome": "pending",
        })
        sixth_path = root / "sixth-slot-ledger.json"
        atomic_json(sixth_path, boundary_ledger)
        if require_root_reservation(ledger_read(sixth_path), boundary_cases[5])["case"] != boundary_cases[5]:
            raise ValueError("six-call boundary rejected its unique final reservation")
        consumed_sixth = _root_ledger_change(sixth_path, boundary_cases[5], "consume", {"logicalChatRequests": 1})
        if consumed_sixth["attemptState"] != "consumed" or ledger_validate(ledger_read(sixth_path))[0] != 6:
            raise ValueError("sixth authorized root slot was not consumed exactly once")
        try:
            _root_ledger_change(sixth_path, boundary_cases[5], "consume", {"logicalChatRequests": 1})
        except ValueError:
            pass
        else:
            raise ValueError("consumed sixth root reservation allowed a duplicate launch")
        try:
            require_root_reservation(ledger_read(sixth_path), boundary_cases[5])
        except ValueError:
            results["rootSlotBoundary"] = "sixth unique slot accepted and consumed once; duplicate launch/consume refused"
        else:
            raise ValueError("consumed sixth root reservation remained eligible for another launch")

        failure_output = root / "skill-failure-prepared"
        prepare(failure_output)
        with redirect_stdout(io.StringIO()):
            command_pin_runtime(argparse.Namespace(prepared=str(failure_output), binary=str(binary)))
        failure_ledger_path = failure_output / "evidence/ledger.json"
        failure_ledger = ledger_read(failure_ledger_path)
        failure_ledger["nativeAuthorization"] = {"state": "authorized", "decision": "offline state-machine regression only; no session launched"}
        atomic_json(failure_ledger_path, failure_ledger)
        skill_root = failure_output / "home/.claude/skills/lds"
        skill_root.mkdir(parents=True)
        (skill_root / "SKILL.md").write_text("offline fake installed skill")
        installed_hash = sha256((skill_root / "SKILL.md").read_bytes())
        _skill_ledger_change(failure_ledger_path, "fresh", "reserve")
        _skill_ledger_change(failure_ledger_path, "fresh", "consume")
        session = {
            "schemaVersion": 1, "channel": "skill", "case": "fresh", "claudeCodeVersion": "2.1.286",
            "provider": "OpenCode Go Anthropic Messages compatibility gateway", "model": "deepseek-v4-flash",
            "sessionExitCode": 9, "operationExitCode": 7, "logicalModelTurns": 99,
            "visibleRetryEvents": "visible-retries:2", "skillInvocation": "/lds", "manualPermissionReview": True,
            "installedSkillSha256": installed_hash, "outcome": "failed",
        }
        session_path = root / "failed-session.json"
        session_path.write_bytes(stable_json(session))
        with redirect_stdout(io.StringIO()):
            command_ledger_settle_skill(argparse.Namespace(prepared=str(failure_output), case="fresh", session_evidence=str(session_path)))
        observation = read_json(failure_output / "evidence/live/skill/fresh/session-observation.json")
        failure_row = ledger_read(failure_ledger_path)["nativeSessions"][0]
        if observation.get("status") != "failed" or observation.get("operationExitCode") != 7 or observation.get("sessionExitCode") != 9 or observation.get("logicalModelTurns") != 99 or observation.get("visibleRetryEvents") != "visible-retries:2" or observation.get("rawStreamsStored") is not False or failure_row.get("outcome") != "failed" or failure_row.get("observedLogicalTurns") != 99 or failure_row.get("authorizationViolation") is not True:
            raise ValueError("failed skill session accounting reset, lost actual counts, or was marked successful")
        try:
            _skill_ledger_change(failure_ledger_path, "partial-engineering", "reserve")
        except ValueError:
            results["nativeFailureAccounting"] = "PASS: 7/9/99 and visible retry count retained; attempt failed, marked over-budget, and blocked later sessions"
        else:
            raise ValueError("native ledger regression: later session allowed after an over-budget failure")

    return results


def command_self_test(_: argparse.Namespace) -> None:
    with redirect_stdout(io.StringIO()):
        command_check_fixtures(argparse.Namespace())
    if canonical_json({"numbers": [333333333.33333329, 1e30, 4.5, 0.002, 1e-27]}) != b'{"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27]}':
        raise ValueError("number serialization self-test failed")
    sample = {"z": [1, True, None], "a": "slash / and newline\n", "\ue000": 1, "\U00010000": 2}
    expected = b'{"a":"slash / and newline\\n","z":[1,true,null],"\xf0\x90\x80\x80":2,"\xee\x80\x80":1}'
    if canonical_json(sample) != expected:
        raise ValueError("UTF-16 ordering/string self-test failed")
    try:
        json.loads('{"a":1,"a":2}', object_pairs_hook=unique_object)
    except ValueError:
        pass
    else:
        raise ValueError("duplicate-property rejection self-test failed")
    with redirect_stdout(io.StringIO()):
        regressions = offline_regressions()
    print(json.dumps({"selfTest": "pass", "duplicateKeys": "rejected", "numericAndUTF16Vectors": "pass", "offlineRegressions": regressions, "providerRequests": 0, "nativeSessions": 0, "mockOnly": True}, sort_keys=True))


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(description=__doc__)
    sub = root.add_subparsers(dest="command", required=True)

    prepare_cmd = sub.add_parser("prepare", help="build paired disposable targets and complete before snapshots")
    prepare_cmd.add_argument("--out", required=True)
    prepare_cmd.add_argument("--profile", choices=sorted(AUTHORIZATION_PROFILES), default="executor")
    prepare_cmd.set_defaults(run=command_prepare)

    pin_cmd = sub.add_parser("pin-runtime", help="pin the offline-built root CLI, helper, fixture assets, and expected target snapshots")
    pin_cmd.add_argument("--prepared", required=True)
    pin_cmd.add_argument("--binary", required=True)
    pin_cmd.set_defaults(run=command_pin_runtime)

    pack_cmd = sub.add_parser("pack-preflight", help="validate and package sanitized zero-inference preflight evidence inside the repository")
    pack_cmd.add_argument("--prepared", required=True)
    pack_cmd.add_argument("--root-cli-binary", required=True)
    pack_cmd.add_argument("--claude-binary", required=True)
    pack_cmd.add_argument("--claude-version", default="2.1.286")
    pack_cmd.add_argument("--out", required=True)
    pack_cmd.add_argument("--offline-only", action="store_true", help="package current offline/mock evidence without claiming root CLI controls were replayed")
    pack_cmd.add_argument("--replace-existing", action="store_true")
    pack_cmd.set_defaults(run=command_pack_preflight)

    fixtures_cmd = sub.add_parser("check-fixtures", help="check independent asset hashes and RFC 8785 vectors")
    fixtures_cmd.set_defaults(run=command_check_fixtures)

    snapshot_cmd = sub.add_parser("snapshot", help="capture every directory/path/type/mode and file byte")
    snapshot_cmd.add_argument("--root", required=True)
    snapshot_cmd.add_argument("--out", required=True)
    snapshot_cmd.set_defaults(run=command_snapshot)

    digest_cmd = sub.add_parser("plan-digest", help="independently verify the digest of a full plan/result JSON")
    digest_cmd.add_argument("--input", required=True)
    digest_cmd.set_defaults(run=command_plan_digest)

    stale_cmd = sub.add_parser("stale-insert", help="insert the known collision only after the live plan/prompt are displayed")
    stale_cmd.add_argument("--prepared", required=True)
    stale_cmd.add_argument("--channel", choices=("root-cli", "skill"), required=True)
    stale_cmd.add_argument("--target", required=True)
    stale_cmd.add_argument("--plan", required=True)
    stale_cmd.set_defaults(run=command_stale_insert)

    run_cmd = sub.add_parser("run-root", help="run real interactive lds adopt on the caller's directly connected terminal")
    run_cmd.add_argument("--prepared", required=True)
    run_cmd.add_argument("--binary", required=True)
    run_cmd.add_argument("--case", choices=sorted(ROOT_CALL_CASES | {"collision", "partial-legacy", "noninteractive"}), required=True)
    run_cmd.add_argument("--channel", choices=("root-cli", "skill"), default="root-cli")
    run_cmd.set_defaults(run=command_run_root)

    verify_cmd = sub.add_parser("verify-root", help="verify sanitized root result plus full before/after snapshots")
    verify_cmd.add_argument("--prepared", required=True)
    verify_cmd.add_argument("--case", choices=("fresh", "rerun", "partial-engineering", "collision", "partial-legacy", "decline", "stale", "override", "noninteractive"), required=True)
    verify_cmd.add_argument("--result", required=True)
    verify_cmd.add_argument("--before", required=True)
    verify_cmd.add_argument("--after", required=True)
    verify_cmd.set_defaults(run=command_verify_outcome)

    control_verify = sub.add_parser("verify-control", help="verify a channel's actual noninteractive no-plan refusal and full unchanged snapshot")
    control_verify.add_argument("--prepared", required=True)
    control_verify.add_argument("--channel", choices=("root-cli", "skill"), required=True)
    control_verify.set_defaults(run=command_verify_control)

    skill_evidence = sub.add_parser("sanitize-skill", help="retain a Claude skill's flow, full plan, apply and snapshot evidence without a raw transcript")
    skill_evidence.add_argument("--prepared", required=True)
    skill_evidence.add_argument("--case", choices=SKILL_CASES, required=True)
    skill_evidence.add_argument("--flow-result", required=True)
    skill_evidence.add_argument("--plan", required=True)
    skill_evidence.add_argument("--before", required=True)
    skill_evidence.add_argument("--tool-results", required=True)
    skill_evidence.add_argument("--session-evidence", required=True)
    skill_evidence.add_argument("--human-checkpoint", required=True)
    skill_evidence.add_argument("--operation-exit-code", type=int, required=True)
    skill_evidence.add_argument("--session-exit-code", type=int, required=True)
    skill_evidence.add_argument("--turns", type=int, required=True)
    skill_evidence.add_argument("--retry-events", required=True, help="sanitized: none, unknown, or visible-retries:<count>; no raw logs")
    skill_evidence.add_argument("--outcome", choices=("applied", "current", "declined", "refused", "failed", "stopped", "not-started"), required=True)
    skill_evidence.add_argument("--claude-version", default="2.1.286")
    skill_evidence.set_defaults(run=command_sanitize_skill)

    skill_verify = sub.add_parser("verify-skill", help="independently verify one collected native skill case and its complete snapshots")
    skill_verify.add_argument("--prepared", required=True)
    skill_verify.add_argument("--case", choices=SKILL_CASES, required=True)
    skill_verify.set_defaults(run=command_verify_skill)

    compare = sub.add_parser("compare-semantic", help="compare root/skill plans, results and complete target snapshots without equating literal digests")
    compare.add_argument("--case", choices=sorted(set(SKILL_CASES)), required=True)
    for option in ("left-result", "right-result", "left-before", "right-before", "left-after", "right-after"):
        compare.add_argument("--" + option, required=True)
    for option in ("left-plan", "right-plan"):
        compare.add_argument("--" + option)
    compare.set_defaults(run=command_compare_semantic)

    ledger_root = sub.add_parser("ledger-reserve-root", help="reserve one eligible root call before manual send consent")
    ledger_root.add_argument("--ledger", required=True)
    ledger_root.add_argument("--case", required=True)
    ledger_root.set_defaults(run=command_ledger_reserve_root)

    ledger_skill = sub.add_parser("ledger-reserve-skill", help="reserve a native session and its full eight-turn ceiling before launch")
    ledger_skill.add_argument("--ledger", required=True)
    ledger_skill.add_argument("--case", required=True)
    ledger_skill.set_defaults(run=command_ledger_reserve_skill)

    ledger_skill_consume = sub.add_parser("ledger-consume-skill", help="consume one authorized native-session reservation before launch")
    ledger_skill_consume.add_argument("--ledger", required=True)
    ledger_skill_consume.add_argument("--case", required=True)
    ledger_skill_consume.set_defaults(run=command_ledger_consume_skill)

    ledger_settle = sub.add_parser("ledger-settle-skill", help="record manually observed native turns and client retry events")
    ledger_settle.add_argument("--prepared", required=True)
    ledger_settle.add_argument("--case", required=True)
    ledger_settle.add_argument("--session-evidence", required=True, help="owner-retained sanitized actual exits/turn/retry counts; never a raw stream")
    ledger_settle.set_defaults(run=command_ledger_settle_skill)

    selftest = sub.add_parser("self-test", help="offline hash, canonicalization, and parser checks")
    selftest.set_defaults(run=command_self_test)
    return root


def main() -> int:
    args = parser().parse_args()
    try:
        args.run(args)
    except (KeyboardInterrupt, RootRunStopped):
        return 130
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"bds251-s10: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
