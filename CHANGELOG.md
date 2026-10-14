# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed
- **Marketplace search.** `GET /api/campaigns?q=` ranks live adverts (active, not ended, not full, budget left) by Atlas Search BM25 on title and description plus meaning (Gemini embeddings, exact cosine over the live set), fused with reciprocal rank fusion. `?eligible=true` keeps only adverts one of the promoter's active social accounts qualifies for. The Earn Hub search box now uses it instead of filtering the loaded page by title, and promoters get an "I qualify" toggle.
- **AI support retrieval is hybrid search.** Learned answers are found with Atlas Vector Search (meaning) and Atlas Search BM25 (keywords), fused with reciprocal rank fusion. Short questions such as "tiktok support?" now find the right answer, and a learned answer never overrides escalation of account-specific problems.
- The API creates its three Atlas Search indexes on start and falls back to in-process ranking on MongoDB without Atlas Search (such as the docker-compose image).
- Embeddings moved to `gemini-embedding-001` at 768 dimensions (`text-embedding-004` was retired); existing learned answers and live campaigns are (re-)embedded on start. Retrieval failures are now logged instead of silently skipped.
- Restructured the repo to the canonical `api/common` / `web` layout.
- Added standard community-health files (CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, CODEOWNERS, issue/PR templates).
