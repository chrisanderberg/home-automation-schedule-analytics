"""Helpers for decoding JSON API response bodies."""

import json
from typing import Any


def decode_json_body(body: str, *, decode_error_as_error_payload: bool) -> dict[str, Any]:
    """Decode a JSON body with configurable fallback on decode errors.

    Args:
        body: Raw response body as UTF-8 text.
        decode_error_as_error_payload: When True, invalid non-empty JSON is
            returned as ``{"error": body}``. When False, invalid JSON is treated
            like an empty body and returns ``{}``.

    Returns:
        Parsed JSON object payload or a fallback dictionary.
    """
    if not body:
        return {}

    try:
        decoded = json.loads(body)
    except json.JSONDecodeError:
        if decode_error_as_error_payload:
            return {"error": body}
        return {}

    if isinstance(decoded, dict):
        return decoded
    return {}
