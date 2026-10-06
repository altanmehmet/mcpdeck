"""Local regressions for Oracle replies that omit MCP isError."""
import unittest

from real_mcp_results import validate_oracle_health, validate_oracle_result


def result(text, **fields):
    return dict(content=[{"type": "text", "text": text}], **fields)


class OracleResultsTest(unittest.TestCase):
    def test_csv_health(self):
        validate_oracle_health(result('"MCPDECK_HEALTH"\n"1"\n'))

    def test_formatted_health(self):
        validate_oracle_health(result("MCPDECK_HEALTH\n-------------\n            1\n"))

    def test_oracle_error_without_is_error(self):
        reply = result("SQLException ORA-00406: private connection details\nORA-00722: feature")
        with self.assertRaisesRegex(AssertionError, r"^Oracle tool failed: ORA-00406, ORA-00722$"):
            validate_oracle_health(reply)

    def test_error_in_second_content(self):
        reply = result("MCPDECK_HEALTH\n1")
        reply["content"].append({"type": "text", "text": "ORA-06512: internal details"})
        with self.assertRaises(AssertionError):
            validate_oracle_result(reply)

    def test_missing_or_wrong_value(self):
        for text in ("Success", "SELECT 1 AS MCPDECK_HEALTH FROM DUAL", "MCPDECK_HEALTH\n0"):
            with self.subTest(text=text), self.assertRaises(AssertionError):
                validate_oracle_health(result(text))

    def test_protocol_error(self):
        with self.assertRaises(AssertionError):
            validate_oracle_result(result("", isError=True))


if __name__ == "__main__":
    unittest.main()
