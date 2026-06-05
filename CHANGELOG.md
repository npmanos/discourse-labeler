# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.1.0 (2026-06-05)


### Features

* add multi-stage Distroless Debian 13 Dockerfile and Compose configurations ([418cfa2](https://github.com/npmanos/discourse-labeler/commit/418cfa2339420ca3a44a1dd352ea993274a046e0))
* add README header, conceptual definition, and quick start ([25dae84](https://github.com/npmanos/discourse-labeler/commit/25dae84ec09412326282cb9c721b37b0a409f55f))
* add release-please manifest configurations, version.go, and initial changelog ([5dcbc9e](https://github.com/npmanos/discourse-labeler/commit/5dcbc9e123f75f9a7fad7f6eb31c81a672bbb659))
* add system prompt override support to Config ([6b77c8f](https://github.com/npmanos/discourse-labeler/commit/6b77c8fff67edf01e6786121557f03f5b21cf6dc))
* define pipeline data models for Jetstream, Slingshot, and LLM ([05010d4](https://github.com/npmanos/discourse-labeler/commit/05010d41a72036c8a380e23a318ee13fd966dd42))
* implement atomic cursor persistence with JSON file storage ([b1c9e4a](https://github.com/npmanos/discourse-labeler/commit/b1c9e4adef684fa9c795f1c2c70d9a5074e67c9f))
* implement daemon entry point and OS signal handling ([cfde31b](https://github.com/npmanos/discourse-labeler/commit/cfde31b2fbbcd885d86760c2dec0b773791ff7ff))
* implement decoupled Ozone API XRPC client and validation ([2d0d11d](https://github.com/npmanos/discourse-labeler/commit/2d0d11d7c3ce10cb99817c5729c9bf7cdb666698))
* implement Graze Contrails WebSocket ingestion client with tests ([f5efacb](https://github.com/npmanos/discourse-labeler/commit/f5efacbaed0f54867f71601c7e761f0bbeba630a))
* implement high-performance concurrent pipeline coordinator and worker pools with tests ([a030b89](https://github.com/npmanos/discourse-labeler/commit/a030b89f4726dbeaa8c5b1b8c696d5271a9dada2))
* implement llama.cpp OpenAI-compatible classification client with tests ([641b064](https://github.com/npmanos/discourse-labeler/commit/641b064f772bc0d625962c90f24e613c62f88257))
* implement Slingshot context hydration client with tests ([5d968bb](https://github.com/npmanos/discourse-labeler/commit/5d968bbb7573c4d2c60081e3ff33d2cd8315319e))
* implement XML post formatter and system prompt overrides in LLMClassifier ([d1c4eac](https://github.com/npmanos/discourse-labeler/commit/d1c4eac008dd9316aae275c1c294cce50af45173))
* initial project setup, git-flow guidelines, and draft spec ([03089c6](https://github.com/npmanos/discourse-labeler/commit/03089c63991ec9e29a508685d463953e9dc94169))
* initialize Go module and environment configuration ([1479b80](https://github.com/npmanos/discourse-labeler/commit/1479b80f1269a3c4ea9d18a20b1dac5f0b8ed594))
* pass LLM system prompt override option in labeler daemon entrypoint ([1ca89fa](https://github.com/npmanos/discourse-labeler/commit/1ca89fa11ec01cb97b32cbdedaa0dbdfdd0e1a06))
* rename emitted Ozone labels to meta-discourse and possible-meta-discourse ([4eeca24](https://github.com/npmanos/discourse-labeler/commit/4eeca24ed7b87bff71903feddf3b2204e799bdaf))
* track agent collaboration rules in repository ([a0392de](https://github.com/npmanos/discourse-labeler/commit/a0392defdb98f30468714e61c839569a05ce8007))


### Bug Fixes

* **config:** propagate system prompt file read errors and add test ([5d6d281](https://github.com/npmanos/discourse-labeler/commit/5d6d281b8ed2c3c7406fcd7f3e5e0172c4577ee2))
* disable fieldalignment in linter and format codebase ([114347c](https://github.com/npmanos/discourse-labeler/commit/114347c44335919349ecb9a28b284784e2a66ecb))
* disable shadow analyzer and resolve goimports issues across services ([0d61062](https://github.com/npmanos/discourse-labeler/commit/0d61062a4ea167ee1214a8927d9a92923de60bac))

## [Unreleased]
