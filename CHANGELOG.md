# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [0.1.0] - 2026-02-10

### Added

- Initial public release
- Query invocations by commit SHA
- List and filter targets (all, failed, flaky)
- Fetch and filter build logs (grep, head, tail)
- List build actions with target label filtering
- Download test artifacts
- Get and delete cached files by bytestream URI
- Execute workflows and remote builds
- Multiple output formats: table, JSON, plaintext, Go template, jq
- Field selection and inline jq filtering
- File output for large responses
- Embedded Claude Code skill (`skill print`, `skill add`)
- Auto-pagination for all list endpoints
