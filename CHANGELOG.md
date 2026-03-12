# Changelog

## [1.10.3](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.10.2...v1.10.3) (2026-03-09)


### Bug Fixes

* auto-persist credentials to config dir when .env is missing ([b52f701](https://gitlab.com/level-87/jirafy-clockwork/-/commit/b52f70149ecbff1ccc48faf109d87fe0d3c94518))

## [1.10.2](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.10.1...v1.10.2) (2026-03-08)


### Bug Fixes

* wire up frontend update UI and fix startup race condition ([a8bc13d](https://gitlab.com/level-87/jirafy-clockwork/-/commit/a8bc13d07c8ee3aaa648703135bb5b0f23cd6860))
* wire up frontend update UI and fix startup race condition ([2d3dfc7](https://gitlab.com/level-87/jirafy-clockwork/-/commit/2d3dfc73ca35913afc8c4715308dc9b59cfb3451))

## [1.10.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.10.0...v1.10.1) (2026-03-08)


### Bug Fixes

* handle Windows drive-letter paths in extractFolderURIs ([db4bbb8](https://gitlab.com/level-87/jirafy-clockwork/-/commit/db4bbb88594bb89481fe05588f2f3cef8faea258))
* light mode and folder access ([73415e9](https://gitlab.com/level-87/jirafy-clockwork/-/commit/73415e9bf52826487e15e24c34efcb2427bf13fe))
* light mode readability for input fields ([f647512](https://gitlab.com/level-87/jirafy-clockwork/-/commit/f647512060718ba258768f81654917ddecbcda46))
* remove VS Code storage.json reading and add protected path filter ([faa33d8](https://gitlab.com/level-87/jirafy-clockwork/-/commit/faa33d8231326570229938dd484746b7007b7152))

## [1.10.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.9.0...v1.10.0) (2026-03-08)


### Features

* auto-update system with beta channel support ([4e473fb](https://gitlab.com/level-87/jirafy-clockwork/-/commit/4e473fbafadc8af5661d60eb3086942978c6cbe7))

## [1.9.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.7...v1.9.0) (2026-03-07)


### Features

* auto-adjust Jira remaining estimate on worklog changes ([bf363fe](https://gitlab.com/level-87/jirafy-clockwork/-/commit/bf363fe074fcd4fd05b3fd5a71663d1db98f0f04))


### Bug Fixes

* delete entry also removes Jira worklog ([b73da55](https://gitlab.com/level-87/jirafy-clockwork/-/commit/b73da550e456b8574bf7f405fba4cee0b293838b))
* replace confirm() with inline delete confirmation ([991fea3](https://gitlab.com/level-87/jirafy-clockwork/-/commit/991fea31981d03c01c3f7b221047a514fc7cfded))
* update entry also syncs Jira worklog from history ([ce6e915](https://gitlab.com/level-87/jirafy-clockwork/-/commit/ce6e91514f412d55ddbf2e552b16b25010b64b37))

## [1.8.7](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.6...v1.8.7) (2026-03-07)


### Bug Fixes

* auto-fetch history on startup and date change ([156b930](https://gitlab.com/level-87/jirafy-clockwork/-/commit/156b930af9336c684d2e60f727f3f6c53b00cd92))

## [1.8.6](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.5...v1.8.6) (2026-03-07)


### Bug Fixes

* persist fetched history in cache, auto-fetch on startup ([e9c7054](https://gitlab.com/level-87/jirafy-clockwork/-/commit/e9c7054534ec3b6dec68daca1f389d9c00dac175))

## [1.8.5](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.4...v1.8.5) (2026-03-07)


### Bug Fixes

* auto-lock ticket on key+space, hide dropdown when editing ([72434e0](https://gitlab.com/level-87/jirafy-clockwork/-/commit/72434e0f05e4dc17b2a0631ee97c7a4a09f4885a))

## [1.8.4](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.3...v1.8.4) (2026-03-07)


### Bug Fixes

* don't search when editing description after ticket selection ([669df16](https://gitlab.com/level-87/jirafy-clockwork/-/commit/669df1629fdba709ca1981c79dcd1e8efbce5124))
* only skip search after key+space, not on exact key match ([c422279](https://gitlab.com/level-87/jirafy-clockwork/-/commit/c4222799cabaeb1236df3ee18ce3043b20d22469))

## [1.8.3](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.2...v1.8.3) (2026-03-07)


### Bug Fixes

* trigger Jira search on first keystroke ([aa191b7](https://gitlab.com/level-87/jirafy-clockwork/-/commit/aa191b7f3deac223d6a7d628dea7bd8885225457))

## [1.8.2](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.1...v1.8.2) (2026-03-07)


### Bug Fixes

* prefix-match across multiple projects, limit dropdown to 5 ([07b16d5](https://gitlab.com/level-87/jirafy-clockwork/-/commit/07b16d565b81981d1c975063f0c9577a7dcc52d0))

## [1.8.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.8.0...v1.8.1) (2026-03-07)


### Bug Fixes

* remove badge, search from 1 char, fully editable description ([161806e](https://gitlab.com/level-87/jirafy-clockwork/-/commit/161806eef34848ae7c0390446412589a161bf0f9))

## [1.8.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.7.1...v1.8.0) (2026-03-07)


### Features

* key prefix search and editable description ([a70bc6b](https://gitlab.com/level-87/jirafy-clockwork/-/commit/a70bc6bf4c2ba66607da482169f771478803caf2))

## [1.7.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.7.0...v1.7.1) (2026-03-07)


### Bug Fixes

* escape JQL special chars and fix input after ticket selection ([c1a4752](https://gitlab.com/level-87/jirafy-clockwork/-/commit/c1a47520b782d639160a4dc4bdce32c76d94af0e))

## [1.7.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.6.2...v1.7.0) (2026-03-07)


### Features

* rework ticket search, fix Jira worklogs, add Clockify projects ([8a3e0f9](https://gitlab.com/level-87/jirafy-clockwork/-/commit/8a3e0f9309719c9936df42a661727a1c81b7999c))

## [1.6.2](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.6.1...v1.6.2) (2026-03-07)


### Bug Fixes

* ticket toggle and dropdown positioning bugs ([5ef77e9](https://gitlab.com/level-87/jirafy-clockwork/-/commit/5ef77e9ed2581d3541f7bb1ae6ff52a5f198211e))
* tray version null — copy data before dispatch_async ([875313b](https://gitlab.com/level-87/jirafy-clockwork/-/commit/875313bf6063ecc7e8572c9803f6847d1fb4a83e))

## [1.6.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.6.0...v1.6.1) (2026-03-07)


### Bug Fixes

* migrate all Jira endpoints to v3 API and fix tray About ([0bd5ea3](https://gitlab.com/level-87/jirafy-clockwork/-/commit/0bd5ea38fe7bd491919fb99df186a44b788fe125))
* set productVersion in wails.json during CI build ([c5309b4](https://gitlab.com/level-87/jirafy-clockwork/-/commit/c5309b41a5372748a0ce8b60a7f7230781962027))

## [1.6.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.5.0...v1.6.0) (2026-03-07)


### Features

* auto theme detection, settings control, macOS tray icon ([ad12164](https://gitlab.com/level-87/jirafy-clockwork/-/commit/ad1216426db2bf5adbf52bc7449d3bc1efea7c22))


### Bug Fixes

* use lowercase JSON property names for workspace dropdown ([2034540](https://gitlab.com/level-87/jirafy-clockwork/-/commit/2034540da5a8631337581e0eea6679915ffcc7c9))

## [1.5.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.4.3...v1.5.0) (2026-03-07)


### Features

* auto-fetch Clockify workspaces from API ([bb2c5fa](https://gitlab.com/level-87/jirafy-clockwork/-/commit/bb2c5faa1adb87559d48b5ac29dd0310c3f23a8e))


### Bug Fixes

* migrate Jira search from deprecated /rest/api/2/search to /rest/api/3/search/jql ([4cd1126](https://gitlab.com/level-87/jirafy-clockwork/-/commit/4cd1126d1ba579ef9a7eb2c47f8e4a67263db790))

## [1.4.3](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.4.2...v1.4.3) (2026-03-07)


### Bug Fixes

* pass secrets to reusable release workflow ([b66085a](https://gitlab.com/level-87/jirafy-clockwork/-/commit/b66085ac3378ef6eff6168ed25e0d8910cb3d295))
* use dedicated HOMEBREW_TAP_TOKEN for homebrew-tap updates ([511b802](https://gitlab.com/level-87/jirafy-clockwork/-/commit/511b80249982d856279bfc8c4e1c4addd8bbe362))

## [1.4.2](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.4.1...v1.4.2) (2026-03-07)


### Bug Fixes

* use RELEASE_PLEASE_TOKEN for homebrew-tap API calls ([d410dfc](https://gitlab.com/level-87/jirafy-clockwork/-/commit/d410dfc57b7df4cafe676f525d52001c75f6f359))

## [1.4.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.4.0...v1.4.1) (2026-03-07)


### Bug Fixes

* use GitHub API for homebrew-tap updates ([7421d51](https://gitlab.com/level-87/jirafy-clockwork/-/commit/7421d51cae0cba7e122bb2ba7a7e804df6dc6d09))

## [1.4.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.3.2...v1.4.0) (2026-03-07)


### Features

* add Homebrew cask for quarantine-free macOS install ([4ae12b0](https://gitlab.com/level-87/jirafy-clockwork/-/commit/4ae12b00aa03ddf3ab58f31c6e0acd6310cec58d))


### Bug Fixes

* save config to user config dir instead of working directory ([257dfa4](https://gitlab.com/level-87/jirafy-clockwork/-/commit/257dfa4c9981ee1109c62673d53f7f24980e9165))

## [1.3.2](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.3.1...v1.3.2) (2026-03-07)


### Bug Fixes

* improve macOS ad-hoc signing with entitlements and hardened runtime ([f8f7b91](https://gitlab.com/level-87/jirafy-clockwork/-/commit/f8f7b9120d4159f11dbf7956cb8bda482f6fb182))

## [1.3.1](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.3.0...v1.3.1) (2026-03-07)


### Bug Fixes

* add ad-hoc code signing and macOS installation instructions ([3512ca0](https://gitlab.com/level-87/jirafy-clockwork/-/commit/3512ca072aa5364e11a9fe8944ba9ae774292446))
* use delete-and-recreate pattern for immutable releases ([da3f96a](https://gitlab.com/level-87/jirafy-clockwork/-/commit/da3f96aaa628fbb3ad4aae17505ea833fedbc4bc))

## [1.3.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.2.0...v1.3.0) (2026-03-06)


### Changes

* No code changes since 1.2.0. This release only updates versioning/release metadata.
## [1.2.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.1.0...v1.2.0) (2026-03-06)


### Features

* **tests:** enhance detector tests with additional cases and mock server integration ([8261f20](https://gitlab.com/level-87/jirafy-clockwork/-/commit/8261f200fb0edef39a0597a3c012d72aa8d7e31d))
* update release-please workflow permissions and enhance documentation for token usage ([8cb41ba](https://gitlab.com/level-87/jirafy-clockwork/-/commit/8cb41bad720f83c40c8da7bd6e5f1414c0b30ec8))


### Bug Fixes

* add CodeQL badge and upgrade to CodeQL v4 ([006becc](https://gitlab.com/level-87/jirafy-clockwork/-/commit/006becc9a1652a44504b00f2932c6426baf3be81))
* decouple release build from release-please workflow ([aaa873d](https://gitlab.com/level-87/jirafy-clockwork/-/commit/aaa873d2a48a243c4d2c9cc5665c160e3531b986))
* make release build resilient to non-critical failures ([93c8600](https://gitlab.com/level-87/jirafy-clockwork/-/commit/93c860074984972793d78d5b4e21d5b461fdec4a))
* rewrite CI/release workflows for reliability ([bfde703](https://gitlab.com/level-87/jirafy-clockwork/-/commit/bfde703ff06fb566d77933e076a754e06c269f92))
* use draft releases to allow asset uploads ([7446d10](https://gitlab.com/level-87/jirafy-clockwork/-/commit/7446d1076908acc0684786ebc44f396cdde868e8))

## [1.1.0](https://gitlab.com/level-87/jirafy-clockwork/-/compare/v1.0.0...v1.1.0) (2026-03-06)


### Features

* **tests:** enhance detector tests with additional cases and mock server integration ([8261f20](https://gitlab.com/level-87/jirafy-clockwork/-/commit/8261f200fb0edef39a0597a3c012d72aa8d7e31d))
* update release-please workflow permissions and enhance documentation for token usage ([8cb41ba](https://gitlab.com/level-87/jirafy-clockwork/-/commit/8cb41bad720f83c40c8da7bd6e5f1414c0b30ec8))


### Bug Fixes

* add CodeQL badge and upgrade to CodeQL v4 ([006becc](https://gitlab.com/level-87/jirafy-clockwork/-/commit/006becc9a1652a44504b00f2932c6426baf3be81))
* rewrite CI/release workflows for reliability ([bfde703](https://gitlab.com/level-87/jirafy-clockwork/-/commit/bfde703ff06fb566d77933e076a754e06c269f92))
