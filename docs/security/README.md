# Security reports

| File | What it is |
| ---- | ---------- |
| [zap-initial.md](zap-initial.md) | OWASP ZAP API scan before the fixes: 0 failures, 2 warnings |
| [zap-final.md](zap-final.md) | The same scan after the fixes: 0 failures, 0 warnings |

Both scans were produced with `zap-api-scan.py` against `docs/openapi.yaml`.
The Trivy scan runs in CI on every push (`.github/workflows/ci.yml`).
