# Safe Redirects

A minimal Chrome extension that rewrites top-level navigations:

| From | To |
| --- | --- |
| `x.com`, `www.x.com` | `nitter.net` |
| `reddit.com`, `www.reddit.com` | `old.reddit.com` |
| `medium.com` and any `*.medium.com` subdomain | `scribe.rip` |

Path, query string, and fragment are preserved.

## Permissions

- `declarativeNetRequestWithHostAccess` — declares the three static redirect rules in `rules.json`. Unlike the plain `declarativeNetRequest` permission, this variant shows no install-time warning and only acts on sites listed in `host_permissions`.
- `host_permissions` — limited to the exact source and destination hosts. No wildcard TLD access, no content scripts, no background page, no tabs/webRequest/storage.

The rules fire only on `main_frame` resources, so subresource requests (images, XHR, iframes, etc.) are untouched.

## Installation (unpacked)

1. Clone or download this directory (`safe-redirects/`) to your machine.
2. Open `chrome://extensions` in Chrome (or any Chromium browser: Edge, Brave, Arc, Vivaldi).
3. Toggle **Developer mode** on (top-right).
4. Click **Load unpacked** and select the `safe-redirects/` directory.
5. Visit `https://x.com/jack` — you should land on `https://nitter.net/jack`.

To update: pull changes, then click the circular reload icon on the extension's card in `chrome://extensions`.

To uninstall: click **Remove** on the extension's card.

## Notes

- Medium subdomain rewrites drop the subdomain (e.g. `uxdesign.medium.com/foo` → `scribe.rip/foo`). Scribe handles the canonical `medium.com/@user/...` form directly; custom publication subdomains are less reliable.
- `old.reddit.com` and `scribe.rip` are excluded from their source rules, so there's no redirect loop.
- Rules are evaluated by Chrome's declarative engine in C++; there is no JavaScript running on your pages.
