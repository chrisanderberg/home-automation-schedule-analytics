"""Unit tests for JSON response-body decode helpers."""

import unittest

from analytics.http_json import decode_json_body


class DecodeJSONBodyTests(unittest.TestCase):
    """Validate JSON decode fallback behavior for API response bodies."""

    def test_parses_valid_json_object_for_both_decode_modes(self):
        """Valid JSON object bodies should decode regardless of fallback mode."""
        body = '{"key":"value"}'
        expected = {"key": "value"}

        for decode_error_as_error_payload in (False, True):
            self.assertEqual(
                decode_json_body(
                    body,
                    decode_error_as_error_payload=decode_error_as_error_payload,
                ),
                expected,
            )

    def test_returns_empty_dict_for_empty_body(self):
        """Empty responses should decode to an empty payload."""
        self.assertEqual(decode_json_body("", decode_error_as_error_payload=False), {})

    def test_returns_empty_dict_for_non_json_when_configured(self):
        """Invalid non-empty JSON should be treated as empty when configured."""
        self.assertEqual(
            decode_json_body("not-json", decode_error_as_error_payload=False),
            {},
        )

    def test_returns_error_payload_for_non_json_when_configured(self):
        """Invalid non-empty JSON should preserve text in an error payload."""
        self.assertEqual(
            decode_json_body("not-json", decode_error_as_error_payload=True),
            {"error": "not-json"},
        )


if __name__ == "__main__":
    unittest.main()
