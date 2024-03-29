# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!--
Added      new features
Changed    changes in existing functionality
Deprecated soon-to-be removed features
Removed    now removed features
Fixed      any bug fixes
Security   in case of vulnerabilities
-->

## [Unreleased]

## [0.5.0] - 2022-08-06
### Added
- Function `ParseArgs` allows you to pass arguments. `Parse` continues to use `os.Args[1:]`.

### Changed
- `fflag.Parse` also parses the given flagset.
- Slightly improved docs, using new godoc features.

## [0.4.0] - 2022-07-31
### Added
- This changelog.

### Removed
- Removed logging functionality. An error is returned instead.


[Unreleased]: https://github.com/nochso/fflag/compare/v5.0.0...HEAD
[0.5.0]: https://github.com/nochso/fflag/compare/v0.5.0...v0.4.0
[0.4.0]: https://github.com/nochso/fflag/compare/v0.4.0...v0.3.1