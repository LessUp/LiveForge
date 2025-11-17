# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive testing framework with unit, integration, e2e, performance, and security tests
- CI/CD pipeline with GitHub Actions
- Code coverage reporting (target ≥80%)
- Load testing tools and performance benchmarks
- Security vulnerability testing
- Project governance files (LICENSE, CONTRIBUTING.md, CODE_OF_CONDUCT.md)
- Issue and pull request templates
- Makefile for build automation
- Docker support with multi-stage builds
- Prometheus metrics integration
- Rate limiting and authentication middleware

### Changed
- Improved error handling and logging throughout the codebase
- Enhanced WebRTC connection management
- Optimized memory usage in media processing
- Updated dependencies to latest secure versions

### Fixed
- Memory leaks in WebRTC peer connection handling
- Race conditions in concurrent stream processing
- Authentication bypass vulnerabilities
- Input validation issues

## [1.0.0] - 2024-01-01

### Added
- Initial release of live-webrtc-go
- WebRTC SFU (Selective Forwarding Unit) implementation
- WHIP/WHEP protocol support
- Recording capabilities
- JWT authentication
- Rate limiting
- Prometheus metrics
- Docker containerization
- Configuration management
- RESTful API endpoints

### Security
- Implemented secure WebRTC connection handling
- Added authentication and authorization
- Rate limiting to prevent abuse
- Input validation and sanitization

---

## Release Notes Template

When creating a new release, use this template:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features

### Changed
- Changes in existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Now removed features

### Fixed
- Bug fixes

### Security
- Security improvements or fixes
```

## Contributing

When adding entries to this changelog:

1. Add entries under the "Unreleased" section
2. Use the appropriate subsection (Added, Changed, Deprecated, Removed, Fixed, Security)
3. Keep entries concise but descriptive
4. Reference issues and pull requests where applicable
5. Group related changes together
6. Use present tense (e.g., "Add" not "Added")

## Versioning

This project uses [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes

Pre-release versions are denoted with a hyphen and identifier (e.g., `2.0.0-beta.1`).