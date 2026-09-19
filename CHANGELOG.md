# Changelog

## [1.3.2](https://github.com/vasu1124/introspect/compare/1.3.1...1.3.2) (2026-09-19)


### Features

* add dynamic configuration support with live ConfigMap synchronization and UI editor ([c86625e](https://github.com/vasu1124/introspect/commit/c86625efdb9c736a8041caada9a0abf5b2dbf58b))
* add WebSocket support for real-time health probe logging and status streaming ([f694287](https://github.com/vasu1124/introspect/commit/f694287010e7d56ccad5a3728738661686661b4e))


### Bug Fixes

* Makefile bug ([d3bdc81](https://github.com/vasu1124/introspect/commit/d3bdc819284aeae2b418b24185d906e630a60c36))


### Miscellaneous Chores

* release 1.3.2 ([1fbcae5](https://github.com/vasu1124/introspect/commit/1fbcae54e820b68b48c32e29af96d14439bf31e3))

## [1.3.1](https://github.com/vasu1124/introspect/compare/1.3.0...1.3.1) (2026-09-18)


### Bug Fixes

* keep the commented pvc ([f97bf9a](https://github.com/vasu1124/introspect/commit/f97bf9a23908dc168e7ce1e5517eadffa1dfcc3f))
* Tiltfile cleanup from slop ([5ab2ab3](https://github.com/vasu1124/introspect/commit/5ab2ab32eb2f32d53db1a40ae27885ec9c348480))


### Miscellaneous Chores

* release 1.3.1 ([d6d6701](https://github.com/vasu1124/introspect/commit/d6d670157ebdb0afa6d57872e1ed8754e70846a3))

## [1.3.0](https://github.com/vasu1124/introspect/compare/1.2.1...1.3.0) (2026-09-17)


### Features

* Add guidelines for AI agents in the Introspect project ([e1e097e](https://github.com/vasu1124/introspect/commit/e1e097efbca63b776672f5ca188f32b77ac4465c))
* add health checks UI and integrate with health endpoints ([b069d4c](https://github.com/vasu1124/introspect/commit/b069d4c6369aaa8536c8ba7df15a3774385e9384))
* add valkey ([8b4c333](https://github.com/vasu1124/introspect/commit/8b4c333168c4a6cc44354a0c3edf4c107049be16))
* Introduce handler interface for consistent route registration ([7c19d2c](https://github.com/vasu1124/introspect/commit/7c19d2cb99b17286d09892f698d5f50d08959959))
* multipurpose Tiltfile ([d847d04](https://github.com/vasu1124/introspect/commit/d847d04079c202e7b27e1bf91f9d7902539af79c))


### Bug Fixes

* clean up code formatting and ensure consistent newline usage across multiple files ([899e407](https://github.com/vasu1124/introspect/commit/899e40721d6c00c570e58def0c1414f40865c986))
* correct status code conversion in MetricsMiddleware and add Hijack support in loggingResponseWriter ([408b71f](https://github.com/vasu1124/introspect/commit/408b71ffa36fca52baff560799225a5d6f309fd2))
* guestbook backend failure modes ([badcc48](https://github.com/vasu1124/introspect/commit/badcc48e100cae13bb009cdf41753db24deda9e1))
* improve backend initialization logging and ensure mutex locking ([ecab027](https://github.com/vasu1124/introspect/commit/ecab0273cb8bebb6326bf721a8921dfa3c7298a6))
* improve toggle switch accessibility and UI consistency in health checks ([2706b8c](https://github.com/vasu1124/introspect/commit/2706b8c58f852dfea7aa483359f2cf34e42c8a59))
* kustomization labels ([c15fc3d](https://github.com/vasu1124/introspect/commit/c15fc3da520482c2d279611d8c35936b27199512))
* multiplatform build ([b460c35](https://github.com/vasu1124/introspect/commit/b460c35184989fe3d5ae78d7942a5010c989e2fc))
* refactor healthz handler to manage separate status codes for live and ready checks ([04e2745](https://github.com/vasu1124/introspect/commit/04e27452120d7620a558d221b709fd96df6201f8))
* replace context.TODO() with context.Background() in Notifier methods ([ba95513](https://github.com/vasu1124/introspect/commit/ba9551337a11614e99cb92ded4608f1a535cbbb4))
* switch to emptyDir ([5f9c6ac](https://github.com/vasu1124/introspect/commit/5f9c6ac29de4b446c6b533eba666a9bc06a5d675))
* tag ([b15cabb](https://github.com/vasu1124/introspect/commit/b15cabbd05de841cb77957006f62d73bcc3436f7))
* update documentation ([7493448](https://github.com/vasu1124/introspect/commit/74934484bf130d703179d04dd266b61df7ef6624))

## [1.2.1](https://github.com/vasu1124/introspect/compare/1.2.0...1.2.1) (2025-11-24)


### Bug Fixes

* kustomization labels ([87ae9c4](https://github.com/vasu1124/introspect/commit/87ae9c48a6be0478044720bb5e27b5456401d81b))
* multiplatform build ([8bc3cf4](https://github.com/vasu1124/introspect/commit/8bc3cf40b9ab2240ab900bd26e54a985649bc2b1))

## [1.2.0](https://github.com/vasu1124/introspect/compare/1.1.0...1.2.0) (2025-11-24)


### Features

* add valkey ([8b4c333](https://github.com/vasu1124/introspect/commit/8b4c333168c4a6cc44354a0c3edf4c107049be16))


### Bug Fixes

* tag ([b15cabb](https://github.com/vasu1124/introspect/commit/b15cabbd05de841cb77957006f62d73bcc3436f7))

## [1.1.0](https://github.com/vasu1124/introspect/compare/v1.0.1...v1.1.0) (2025-11-23)


### Features

* add valkey ([8b4c333](https://github.com/vasu1124/introspect/commit/8b4c333168c4a6cc44354a0c3edf4c107049be16))


### Bug Fixes

* tag ([b15cabb](https://github.com/vasu1124/introspect/commit/b15cabbd05de841cb77957006f62d73bcc3436f7))

## [1.0.1](https://github.com/vasu1124/introspect/compare/v1.0.0...v1.0.1) (2025-11-22)


### Bug Fixes

* tag ([b15cabb](https://github.com/vasu1124/introspect/commit/b15cabbd05de841cb77957006f62d73bcc3436f7))
