# Pulse Investor Pitch Site

A single-page pitch site for Pulse, built plain HTML/CSS/JS, no build step or framework. Content is drawn from `docs/PROJECT_SUMMARY.md` (one level up) and the pitch Q&A worked out for Pulse's pre-seed funding application.

## Structure

```
pitch/
├── index.html      All page content and sections
├── css/style.css   Styling (matches Pulse's brand blue #2563EB)
├── js/main.js      Mobile nav toggle + scroll reveal animation
├── render.yaml     Render static site config (reference, see note below)
└── README.md
```

## Local preview

Just open `index.html` in a browser, or serve it locally:

```bash
npx serve .
```

## Deploying to Render

This lives inside the main `pulse` monorepo, so Render needs to be told to build only this subfolder.

1. On [render.com](https://render.com), go to New, then Static Site, and connect the `pulse` repo.
2. Under Advanced settings, set:
   - **Root Directory:** `pitch`
   - **Build Command:** (leave empty)
   - **Publish Directory:** `.`
3. Deploy. Render gives you a free `*.onrender.com` URL right away, a custom domain can be attached later under Settings, then Custom Domains.

`render.yaml` in this folder documents the same settings for reference, but Render's Blueprint auto-detection only scans the repo root by default, so set Root Directory manually as above rather than relying on it.

## Content source of truth

Numbers quoted on the page (raise amount, valuation, market size, traction) reflect what was agreed as of 2026-09-13. If any of these change, update `index.html` directly, sections are commented and named (`#problem`, `#solution`, `#market`, `#model`, `#traction`, `#team`, `#ask`).
