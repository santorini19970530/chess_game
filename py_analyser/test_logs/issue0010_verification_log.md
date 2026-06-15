# issue0010 Verification Log

Date: 2026-06-15

## 1) Test code (automated)

- File: `py_analyser/test_issue0010_service.py`
- Command:
  - `"./.venv/bin/python" "./py_analyser/test_issue0010_service.py"`
- Result: PASS
- Output log: `py_analyser/test_logs/issue0010_unittest.log`

Observed output:

```text
----------------------------------------------------------------------
Ran 3 tests in 0.053s

OK
```

## 2) Manual API checks (saved)

- `GET /health`
  - Saved raw HTTP response: `py_analyser/test_logs/issue0010_health_http.txt`
  - Status: `HTTP/1.1 200 OK`
  - JSON body includes `service`, `status`, `timestamp`

- `POST /analyze` with sample FEN
  - Saved raw HTTP response: `py_analyser/test_logs/issue0010_analyze_http.txt`
  - Status: `HTTP/1.1 200 OK`
  - JSON body includes required fields:
    - `request_id`, `status`, `source`, `fen`, `evaluated_for_color`
    - `health_summary`, `eval_cp_white`
    - `win_chance_white`, `win_chance_black`
    - `best_move_uci`, `suggested_moves`, `latency_ms`

## 3) Go async worker evidence

- Move command sent to Go backend and saved:
  - `py_analyser/test_logs/issue0010_move_response.json`
- Go backend terminal output shows async worker analyzer log:
  - `analyzer response: {...}`
  - includes `request_id`, `status`, `health_summary`, `best_move_uci`, `suggested_moves`, and win chance fields.

