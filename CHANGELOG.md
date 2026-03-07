# Changelog

## [1.6.1](https://github.com/emmesbef/clockify-jira-sync/compare/v1.6.0...v1.6.1) (2026-03-07)


### Bug Fixes

* migrate all Jira endpoints to v3 API and fix tray About ([0bd5ea3](https://github.com/emmesbef/clockify-jira-sync/commit/0bd5ea38fe7bd491919fb99df186a44b788fe125))
* set productVersion in wails.json during CI build ([c5309b4](https://github.com/emmesbef/clockify-jira-sync/commit/c5309b41a5372748a0ce8b60a7f7230781962027))

## [1.6.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.5.0...v1.6.0) (2026-03-07)


### Features

* auto theme detection, settings control, macOS tray icon ([ad12164](https://github.com/emmesbef/clockify-jira-sync/commit/ad1216426db2bf5adbf52bc7449d3bc1efea7c22))


### Bug Fixes

* use lowercase JSON property names for workspace dropdown ([2034540](https://github.com/emmesbef/clockify-jira-sync/commit/2034540da5a8631337581e0eea6679915ffcc7c9))

## [1.5.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.4.3...v1.5.0) (2026-03-07)


### Features

* auto-fetch Clockify workspaces from API ([bb2c5fa](https://github.com/emmesbef/clockify-jira-sync/commit/bb2c5faa1adb87559d48b5ac29dd0310c3f23a8e))


### Bug Fixes

* migrate Jira search from deprecated /rest/api/2/search to /rest/api/3/search/jql ([4cd1126](https://github.com/emmesbef/clockify-jira-sync/commit/4cd1126d1ba579ef9a7eb2c47f8e4a67263db790))

## [1.4.3](https://github.com/emmesbef/clockify-jira-sync/compare/v1.4.2...v1.4.3) (2026-03-07)


### Bug Fixes

* pass secrets to reusable release workflow ([b66085a](https://github.com/emmesbef/clockify-jira-sync/commit/b66085ac3378ef6eff6168ed25e0d8910cb3d295))
* use dedicated HOMEBREW_TAP_TOKEN for homebrew-tap updates ([511b802](https://github.com/emmesbef/clockify-jira-sync/commit/511b80249982d856279bfc8c4e1c4addd8bbe362))

## [1.4.2](https://github.com/emmesbef/clockify-jira-sync/compare/v1.4.1...v1.4.2) (2026-03-07)


### Bug Fixes

* use RELEASE_PLEASE_TOKEN for homebrew-tap API calls ([d410dfc](https://github.com/emmesbef/clockify-jira-sync/commit/d410dfc57b7df4cafe676f525d52001c75f6f359))

## [1.4.1](https://github.com/emmesbef/clockify-jira-sync/compare/v1.4.0...v1.4.1) (2026-03-07)


### Bug Fixes

* use GitHub API for homebrew-tap updates ([7421d51](https://github.com/emmesbef/clockify-jira-sync/commit/7421d51cae0cba7e122bb2ba7a7e804df6dc6d09))

## [1.4.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.3.2...v1.4.0) (2026-03-07)


### Features

* add Homebrew cask for quarantine-free macOS install ([4ae12b0](https://github.com/emmesbef/clockify-jira-sync/commit/4ae12b00aa03ddf3ab58f31c6e0acd6310cec58d))


### Bug Fixes

* save config to user config dir instead of working directory ([257dfa4](https://github.com/emmesbef/clockify-jira-sync/commit/257dfa4c9981ee1109c62673d53f7f24980e9165))

## [1.3.2](https://github.com/emmesbef/clockify-jira-sync/compare/v1.3.1...v1.3.2) (2026-03-07)


### Bug Fixes

* improve macOS ad-hoc signing with entitlements and hardened runtime ([f8f7b91](https://github.com/emmesbef/clockify-jira-sync/commit/f8f7b9120d4159f11dbf7956cb8bda482f6fb182))

## [1.3.1](https://github.com/emmesbef/clockify-jira-sync/compare/v1.3.0...v1.3.1) (2026-03-07)


### Bug Fixes

* add ad-hoc code signing and macOS installation instructions ([3512ca0](https://github.com/emmesbef/clockify-jira-sync/commit/3512ca072aa5364e11a9fe8944ba9ae774292446))
* use delete-and-recreate pattern for immutable releases ([da3f96a](https://github.com/emmesbef/clockify-jira-sync/commit/da3f96aaa628fbb3ad4aae17505ea833fedbc4bc))

## [1.3.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.2.0...v1.3.0) (2026-03-06)


### Changes

* No code changes since 1.2.0. This release only updates versioning/release metadata.
## [1.2.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.1.0...v1.2.0) (2026-03-06)


### Features

* **tests:** enhance detector tests with additional cases and mock server integration ([8261f20](https://github.com/emmesbef/clockify-jira-sync/commit/8261f200fb0edef39a0597a3c012d72aa8d7e31d))
* update release-please workflow permissions and enhance documentation for token usage ([8cb41ba](https://github.com/emmesbef/clockify-jira-sync/commit/8cb41bad720f83c40c8da7bd6e5f1414c0b30ec8))


### Bug Fixes

* add CodeQL badge and upgrade to CodeQL v4 ([006becc](https://github.com/emmesbef/clockify-jira-sync/commit/006becc9a1652a44504b00f2932c6426baf3be81))
* decouple release build from release-please workflow ([aaa873d](https://github.com/emmesbef/clockify-jira-sync/commit/aaa873d2a48a243c4d2c9cc5665c160e3531b986))
* make release build resilient to non-critical failures ([93c8600](https://github.com/emmesbef/clockify-jira-sync/commit/93c860074984972793d78d5b4e21d5b461fdec4a))
* rewrite CI/release workflows for reliability ([bfde703](https://github.com/emmesbef/clockify-jira-sync/commit/bfde703ff06fb566d77933e076a754e06c269f92))
* use draft releases to allow asset uploads ([7446d10](https://github.com/emmesbef/clockify-jira-sync/commit/7446d1076908acc0684786ebc44f396cdde868e8))

## [1.1.0](https://github.com/emmesbef/clockify-jira-sync/compare/v1.0.0...v1.1.0) (2026-03-06)


### Features

* **tests:** enhance detector tests with additional cases and mock server integration ([8261f20](https://github.com/emmesbef/clockify-jira-sync/commit/8261f200fb0edef39a0597a3c012d72aa8d7e31d))
* update release-please workflow permissions and enhance documentation for token usage ([8cb41ba](https://github.com/emmesbef/clockify-jira-sync/commit/8cb41bad720f83c40c8da7bd6e5f1414c0b30ec8))


### Bug Fixes

* add CodeQL badge and upgrade to CodeQL v4 ([006becc](https://github.com/emmesbef/clockify-jira-sync/commit/006becc9a1652a44504b00f2932c6426baf3be81))
* rewrite CI/release workflows for reliability ([bfde703](https://github.com/emmesbef/clockify-jira-sync/commit/bfde703ff06fb566d77933e076a754e06c269f92))
