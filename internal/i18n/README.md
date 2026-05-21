# UI locales (`internal/i18n`)

Embedded JSON under [`locales/`](locales/). Loaded at startup; **English** is the fallback for missing keys.

**Operator and contributor guide:** see [Web UI languages (i18n)](../../README.md#web-ui-languages-i18n) in the root README.

**CI:** `go test ./internal/i18n/...` checks that every enabled locale file has the same keys as `en.json`.
