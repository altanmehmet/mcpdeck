"""Validate Oracle smoke results without exposing server response content."""
import csv
import re


def validate_oracle_result(result):
    if result.get("isError"):
        raise AssertionError("Oracle tool reported an error")
    text = "\n".join(item.get("text", "") for item in result.get("content", [])
                     if item.get("type") == "text")
    errors = sorted(set(re.findall(r"\b(?:ORA|TNS|SP2)-\d+\b", text)))
    if errors:
        raise AssertionError("Oracle tool failed: " + ", ".join(errors))
    return text


def validate_oracle_health(result):
    text = validate_oracle_result(result)
    # SQLcl returns CSV or a single-column formatted SQL result. Require the
    # actual column and its constant value, not merely a successful MCP envelope.
    rows = list(csv.reader(text.splitlines()))
    for index, row in enumerate(rows):
        if len(row) != 1 or row[0].strip() != "MCPDECK_HEALTH":
            continue
        for value in rows[index + 1:]:
            if not value or (len(value) == 1 and re.fullmatch(r"[\s-]*", value[0])):
                continue
            if len(value) == 1 and value[0].strip() == "1":
                return
            break
    raise AssertionError("Oracle health result did not contain the expected value 1")
