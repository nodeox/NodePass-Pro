#!/usr/bin/env python3
"""NodePass Pro 授权验证工具。

用途:
1. 调用远程授权验证接口
2. 校验返回的版本策略(min/max)
3. 以 JSON 输出标准化结果
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from urllib.parse import urlparse
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass
class Versions:
    panel: str
    backend: str
    frontend: str
    nodeclient: str


def fixed_verify_url() -> str:
    scheme = "https://"
    host = "key." + "hahaha." + "ooo"
    path = "/api/v1/verify"
    return f"{scheme}{host}{path}"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="NodePass Pro 授权验证")
    parser.add_argument("--verify-url", default="", help="授权验证接口地址（默认内置地址）")
    parser.add_argument("--license-key", required=True, help="授权码")
    parser.add_argument("--machine-id", required=True, help="机器唯一标识")
    parser.add_argument("--action", required=True, choices=["install", "upgrade"], help="动作类型")
    parser.add_argument("--panel-version", required=True, help="panel 版本")
    parser.add_argument("--backend-version", required=True, help="backend 版本")
    parser.add_argument("--frontend-version", required=True, help="frontend 版本")
    parser.add_argument("--nodeclient-version", required=True, help="nodeclient 版本")
    parser.add_argument("--branch", default="", help="仓库分支")
    parser.add_argument("--commit", default="", help="仓库提交")
    parser.add_argument("--domain", default="", help="授权绑定域名")
    parser.add_argument("--site-url", default="", help="授权绑定站点地址")
    parser.add_argument("--channel", default="stable", help="版本通道（统一接口使用）")
    parser.add_argument("--timeout", type=int, default=20, help="请求超时秒数")
    return parser.parse_args()


def version_tuple(raw: str) -> tuple[int, int, int, str]:
    value = (raw or "").strip()
    value = value.lstrip("vV")
    match = re.match(r"^(\d+)\.(\d+)\.(\d+)(.*)$", value)
    if not match:
        return (0, 0, 0, value)
    major, minor, patch, suffix = match.groups()
    return (int(major), int(minor), int(patch), suffix or "")


def compare_semver(current: str, target: str) -> int:
    left = version_tuple(current)
    right = version_tuple(target)
    if left[:3] < right[:3]:
        return -1
    if left[:3] > right[:3]:
        return 1

    if left[3] == right[3]:
        return 0
    if left[3] == "":
        return 1
    if right[3] == "":
        return -1
    if left[3] < right[3]:
        return -1
    if left[3] > right[3]:
        return 1
    return 0


def get_nested(data: dict[str, Any], *paths: str) -> Any:
    for path in paths:
        current: Any = data
        ok = True
        for key in path.split("."):
            if not isinstance(current, dict) or key not in current:
                ok = False
                break
            current = current[key]
        if ok:
            return current
    return None


def normalize_policy(policy_raw: Any) -> dict[str, str]:
    if not isinstance(policy_raw, dict):
        return {}

    mapping = {
        "min_panel_version": ["min_panel_version", "minPanelVersion"],
        "max_panel_version": ["max_panel_version", "maxPanelVersion"],
        "min_backend_version": ["min_backend_version", "minBackendVersion"],
        "max_backend_version": ["max_backend_version", "maxBackendVersion"],
        "min_frontend_version": ["min_frontend_version", "minFrontendVersion"],
        "max_frontend_version": ["max_frontend_version", "maxFrontendVersion"],
        "min_nodeclient_version": ["min_nodeclient_version", "minNodeclientVersion"],
        "max_nodeclient_version": ["max_nodeclient_version", "maxNodeclientVersion"],
    }

    normalized: dict[str, str] = {}
    for canonical, aliases in mapping.items():
        for alias in aliases:
            value = policy_raw.get(alias)
            if isinstance(value, str) and value.strip():
                normalized[canonical] = value.strip()
                break
    return normalized


def check_version_policy(versions: Versions, policy: dict[str, str]) -> tuple[bool, str]:
    checks = [
        ("panel", versions.panel, policy.get("min_panel_version"), policy.get("max_panel_version")),
        ("backend", versions.backend, policy.get("min_backend_version"), policy.get("max_backend_version")),
        ("frontend", versions.frontend, policy.get("min_frontend_version"), policy.get("max_frontend_version")),
        ("nodeclient", versions.nodeclient, policy.get("min_nodeclient_version"), policy.get("max_nodeclient_version")),
    ]

    for name, current, min_version, max_version in checks:
        if min_version and compare_semver(current, min_version) < 0:
            return False, f"{name} 版本过低: current={current}, min={min_version}"
        if max_version and compare_semver(current, max_version) > 0:
            return False, f"{name} 版本过高: current={current}, max={max_version}"
    return True, "版本策略校验通过"


def do_verify(args: argparse.Namespace) -> dict[str, Any]:
    verify_url = resolve_verify_url(args.verify_url)
    if is_unified_verify_url(verify_url):
        return do_verify_unified(args, verify_url)
    return do_verify_legacy(args, verify_url)


def resolve_verify_url(raw: str) -> str:
    value = (raw or "").strip()
    if value:
        return value
    return fixed_verify_url()


def is_unified_verify_url(verify_url: str) -> bool:
    try:
        parsed = urlparse(verify_url)
    except ValueError:
        return False
    path = (parsed.path or "").rstrip("/")
    return path.endswith("/api/v1/verify")


def post_json(url: str, payload: dict[str, Any], timeout: int) -> tuple[int, dict[str, Any] | None, str]:
    request = urllib.request.Request(
        url,
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            raw = response.read().decode("utf-8")
            code = getattr(response, "status", 200)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        return exc.code, None, body
    except Exception:  # noqa: BLE001
        return 0, None, "请求授权接口失败"

    try:
        data = json.loads(raw)
    except Exception as exc:  # noqa: BLE001
        return code, None, f"授权接口返回非 JSON: {exc}"
    return code, data, raw


def do_verify_unified(args: argparse.Namespace, verify_url: str) -> dict[str, Any]:
    domain = (args.domain or "").strip()
    site_url = (args.site_url or "").strip()

    component_versions = [
        ("panel", args.panel_version),
        ("backend", args.backend_version),
        ("frontend", args.frontend_version),
        ("nodeclient", args.nodeclient_version),
    ]

    detail: dict[str, Any] = {}
    license_snapshot: dict[str, Any] = {}

    for product, current_version in component_versions:
        payload: dict[str, Any] = {
            "license_key": args.license_key,
            "machine_id": args.machine_id,
            "hostname": args.machine_id,
            "product": product,
            "client_version": current_version,
            "channel": args.channel,
        }
        # 兼容保留字段（统一接口会忽略未知字段）
        payload["action"] = args.action
        if domain:
            payload["domain"] = domain
        if site_url:
            payload["site_url"] = site_url

        status_code, data, raw_or_err = post_json(verify_url, payload, args.timeout)
        if status_code == 0:
            return {
                "valid": False,
                "message": raw_or_err,
            }
        if status_code < 200 or status_code >= 300:
            return {
                "valid": False,
                "message": f"授权接口返回 HTTP {status_code}",
                "error": raw_or_err,
            }
        if data is None:
            return {
                "valid": False,
                "message": raw_or_err,
            }

        success = bool(data.get("success", False))
        payload_data = data.get("data") or {}
        verified = bool(payload_data.get("verified", False))
        license_data = payload_data.get("license") or {}
        version_data = payload_data.get("version") or {}

        detail[product] = {
            "verified": verified,
            "status": payload_data.get("status"),
            "license_status": license_data.get("status"),
            "version_status": version_data.get("status"),
            "message": version_data.get("message") or license_data.get("message") or payload_data.get("status"),
            "latest_version": version_data.get("latest_version"),
        }

        if not license_snapshot:
            license_snapshot = {
                "license_id": license_data.get("license_id"),
                "customer": license_data.get("customer"),
                "plan": license_data.get("plan_code"),
                "expires_at": license_data.get("expires_at"),
            }

        if not success or not verified:
            fail_msg = (
                version_data.get("message")
                or license_data.get("message")
                or payload_data.get("status")
                or data.get("message")
                or f"{product} 校验失败"
            )
            return {
                "valid": False,
                "message": str(fail_msg),
                "component": product,
                "detail": detail,
                **license_snapshot,
                "data": data,
            }

    return {
        "valid": True,
        "message": "授权与版本校验通过",
        **license_snapshot,
        "component_results": detail,
    }


def do_verify_legacy(args: argparse.Namespace, verify_url: str) -> dict[str, Any]:
    domain = (args.domain or "").strip()
    site_url = (args.site_url or "").strip()

    payload = {
        "license_key": args.license_key,
        "machine_id": args.machine_id,
        "action": args.action,
        "versions": {
            "panel": args.panel_version,
            "backend": args.backend_version,
            "frontend": args.frontend_version,
            "nodeclient": args.nodeclient_version,
        },
        "branch": args.branch,
        "commit": args.commit,
    }
    if domain:
        payload["domain"] = domain
    if site_url:
        payload["site_url"] = site_url

    request = urllib.request.Request(
        verify_url,
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(request, timeout=args.timeout) as response:
            raw = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        return {
            "valid": False,
            "message": f"授权接口返回 HTTP {exc.code}",
            "error": body,
        }
    except Exception as exc:  # noqa: BLE001
        _ = exc  # 不回传底层异常，避免泄露内部地址
        return {
            "valid": False,
            "message": "请求授权接口失败",
        }

    try:
        data = json.loads(raw)
    except Exception as exc:  # noqa: BLE001
        return {
            "valid": False,
            "message": f"授权接口返回非 JSON: {exc}",
            "raw": raw,
        }

    valid = get_nested(data, "data.valid", "valid")
    message = get_nested(data, "data.message", "message")
    if isinstance(valid, bool) and not valid:
        return {
            "valid": False,
            "message": str(message or "授权校验失败"),
            "data": data,
        }

    if valid is None:
        success = get_nested(data, "success")
        if success is False:
            return {
                "valid": False,
                "message": str(message or "授权接口返回失败"),
                "data": data,
            }
        valid = True

    policy = normalize_policy(get_nested(data, "data.version_policy", "version_policy"))
    versions = Versions(
        panel=args.panel_version,
        backend=args.backend_version,
        frontend=args.frontend_version,
        nodeclient=args.nodeclient_version,
    )
    policy_ok, policy_message = check_version_policy(versions, policy)
    if not policy_ok:
        return {
            "valid": False,
            "message": policy_message,
            "version_policy": policy,
            "data": data,
        }

    return {
        "valid": True,
        "message": str(message or policy_message or "授权校验通过"),
        "license_id": get_nested(data, "data.license_id", "license_id", "data.licenseId", "licenseId"),
        "customer": get_nested(data, "data.customer", "customer"),
        "plan": get_nested(data, "data.plan", "plan"),
        "expires_at": get_nested(data, "data.expires_at", "expires_at", "data.expiresAt", "expiresAt"),
        "version_policy": policy,
        "data": data,
    }


def main() -> None:
    args = parse_args()
    result = do_verify(args)
    print(json.dumps(result, ensure_ascii=False))
    if result.get("valid") is True:
        sys.exit(0)
    sys.exit(2)


if __name__ == "__main__":
    main()
