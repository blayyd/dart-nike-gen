# Dart IO Nike Gen

Dart automates Nike account creation for raffle entries. It drives a Chromium browser through Nike's registration flow, handles SMS phone verification through two provider integrations, rotates proxies per session, and posts results to a Discord webhook.

![1723754023145](https://github.com/user-attachments/assets/f19e6064-fdad-4f83-b74e-75bfa1d9e9e6)

---

## Techniques

- **Goroutine pool with bounded concurrency** — [`main.go`](main.go) uses [ants](https://github.com/panjf2000/ants) to cap parallel browser sessions at a user-set thread count. A `sync.WaitGroup` tracks task completion so the process exits cleanly once the account target is reached.

- **Chrome DevTools Protocol event interception** — The tool listens for `fetch.EventAuthRequired` and `fetch.EventRequestPaused` CDP events to inject proxy credentials mid-request. This avoids patching the browser binary or relying on environment-level proxy settings.

- **Mobile device emulation** — `chromedp.Emulate(device.IPhone11Pro)` applies the full iPhone 11 Pro profile (user-agent, viewport, and touch events) to the browser context. Nike's registration endpoint behaves differently for mobile clients.

- **Context-scoped browser lifecycle** — Each account task gets its own `chromedp.NewExecAllocator` context with a 6-minute `context.WithTimeout`. When the timeout fires or the task finishes, the context cancellation tears down the browser process automatically.

- **[XPath](https://developer.mozilla.org/en-US/docs/Web/XPath)-based DOM targeting** — Registration form fields are located with absolute XPath expressions passed to `chromedp.SendKeys` and `chromedp.Click`. No CSS selector parsing is needed.

- **Cryptographically shuffled password generation** — Passwords are built by placing a guaranteed digit and special character at known indices, filling the rest from a combined character set, then randomizing all positions with `rand.Shuffle`. Every generated password satisfies Nike's complexity rules.

- **Function-scoped struct definitions** — `Config`, `Active`, and `Order` JSON-mapping structs are declared inside `process()` rather than at package scope. They stay local to the function and don't pollute the package namespace.

- **Closure-based task units** — The registration flow is wrapped in a closure submitted to the goroutine pool. Per-task state (proxy, credentials, phone number) stays inside the closure without shared mutable state.

---

## Libraries

| Library | Purpose |
|---|---|
| [chromedp](https://github.com/chromedp/chromedp) | Chrome DevTools Protocol driver for browser automation |
| [panjf2000/ants](https://github.com/panjf2000/ants) | Reusable goroutine pool with capacity limits |
| [Pallinder/go-randomdata](https://github.com/Pallinder/go-randomdata) | Generates random names and numbers for account identities |
| [hugolgst/rich-go](https://github.com/hugolgst/rich-go) | Discord Rich Presence client — displays live status on your Discord profile |
| [lxi1400/GoTitle](https://github.com/lxi1400/GoTitle) | Sets the terminal window title cross-platform |
| [fatih/color](https://github.com/fatih/color) | ANSI color output for terminal logging |
| [go-rod/bypass](https://github.com/go-rod/bypass) | Patches Chromium to evade common bot-detection fingerprints |

SMS verification is handled through either [sms-activate.ru](https://sms-activate.ru) or [sms.discount](https://sms.discount), selectable at runtime.

---

## Project Structure

```
dart-nike-gen/
├── main.go
├── config.json
├── go.mod
├── go.sum
├── proxies.txt
└── accounts.txt
```

**`proxies.txt`** — User-supplied. One proxy per line in `ip:port` or `ip:port:user:pass` format. Required before running.

**`accounts.txt`** — Generated output. Written on each successful registration; created automatically on first success.
