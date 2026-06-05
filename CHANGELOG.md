# 0.1.0 (2026-06-05)


### Bug Fixes

* add a new few-shot example to LLMClassifier for Bluesky friends context ([4adafc8](https://github.com/npmanos/discourse-labeler/commit/4adafc83833ca3c7f482ffb6e88572cee4a828bf))
* **config:** propagate system prompt file read errors and add test ([5d6d281](https://github.com/npmanos/discourse-labeler/commit/5d6d281b8ed2c3c7406fcd7f3e5e0172c4577ee2))
* disable fieldalignment in linter and format codebase ([114347c](https://github.com/npmanos/discourse-labeler/commit/114347c44335919349ecb9a28b284784e2a66ecb))
* disable shadow analyzer and resolve goimports issues across services ([0d61062](https://github.com/npmanos/discourse-labeler/commit/0d61062a4ea167ee1214a8927d9a92923de60bac))
* populate target post CID in Ozone emitEvent payloads ([20b2fba](https://github.com/npmanos/discourse-labeler/commit/20b2fba46ea149cb4f4f87d46aa59e3fa7682f7e))
* support Jetstream kind and operation JSON mappings in RawEvent and JetstreamCommit ([901fb67](https://github.com/npmanos/discourse-labeler/commit/901fb676c561c083b3a3c97a399f6707af4164c1))
* update LLM GGUF model filename in docker-compose ([2e64cd2](https://github.com/npmanos/discourse-labeler/commit/2e64cd2f6ef118bdee2c09e1ff11ba57e1c59534))
* update Ozone client authentication to use HTTP Basic Auth ([d97b84b](https://github.com/npmanos/discourse-labeler/commit/d97b84b40fc6a5fb6df89cc5daadb929f06fa19e))


### Features

* add classification result and dry run label emission debug logging ([7e3ca39](https://github.com/npmanos/discourse-labeler/commit/7e3ca39ab863a39f1db2c2e1f3a65a7cfaede2c2))
* add multi-stage Distroless Debian 13 Dockerfile and Compose configurations ([418cfa2](https://github.com/npmanos/discourse-labeler/commit/418cfa2339420ca3a44a1dd352ea993274a046e0))
* add PostClassification and ContextAnalysis structs to pipeline types ([c22ce32](https://github.com/npmanos/discourse-labeler/commit/c22ce32ac829035c96832b6b771b57ed410c277c))
* add README header, conceptual definition, and quick start ([25dae84](https://github.com/npmanos/discourse-labeler/commit/25dae84ec09412326282cb9c721b37b0a409f55f))
* add release-please manifest configurations, version.go, and initial changelog ([5dcbc9e](https://github.com/npmanos/discourse-labeler/commit/5dcbc9e123f75f9a7fad7f6eb31c81a672bbb659))
* add support for optional LLM_API_KEY environment variable ([bda3e4d](https://github.com/npmanos/discourse-labeler/commit/bda3e4d0b77715dc5686d960fd2e7ae0d6ce1ce7))
* add system prompt override support to Config ([6b77c8f](https://github.com/npmanos/discourse-labeler/commit/6b77c8fff67edf01e6786121557f03f5b21cf6dc))
* define pipeline data models for Jetstream, Slingshot, and LLM ([05010d4](https://github.com/npmanos/discourse-labeler/commit/05010d41a72036c8a380e23a318ee13fd966dd42))
* implement atomic cursor persistence with JSON file storage ([b1c9e4a](https://github.com/npmanos/discourse-labeler/commit/b1c9e4adef684fa9c795f1c2c70d9a5074e67c9f))
* implement categorical routing and rich text debugging in coordinator ([a159d05](https://github.com/npmanos/discourse-labeler/commit/a159d05a08eb7a073c9836c6aae508a05a29a0ea))
* implement daemon entry point and OS signal handling ([cfde31b](https://github.com/npmanos/discourse-labeler/commit/cfde31b2fbbcd885d86760c2dec0b773791ff7ff))
* implement decoupled Ozone API XRPC client and validation ([2d0d11d](https://github.com/npmanos/discourse-labeler/commit/2d0d11d7c3ce10cb99817c5729c9bf7cdb666698))
* implement Graze Contrails WebSocket ingestion client with tests ([f5efacb](https://github.com/npmanos/discourse-labeler/commit/f5efacbaed0f54867f71601c7e761f0bbeba630a))
* implement high-performance concurrent pipeline coordinator and worker pools with tests ([a030b89](https://github.com/npmanos/discourse-labeler/commit/a030b89f4726dbeaa8c5b1b8c696d5271a9dada2))
* implement llama.cpp OpenAI-compatible classification client with tests ([641b064](https://github.com/npmanos/discourse-labeler/commit/641b064f772bc0d625962c90f24e613c62f88257))
* implement Ozone EmitEscalation and rich comment formatting ([88a0234](https://github.com/npmanos/discourse-labeler/commit/88a0234d0e1438400f085f7a8c4df682ba823fcb))
* implement revised system prompt, JSON schema, few-shots, and logprob calculation ([262bd6b](https://github.com/npmanos/discourse-labeler/commit/262bd6b0c12bace34050d10cb9df986bd04ec73c))
* implement Slingshot context hydration client with tests ([5d968bb](https://github.com/npmanos/discourse-labeler/commit/5d968bbb7573c4d2c60081e3ff33d2cd8315319e))
* implement XML post formatter and system prompt overrides in LLMClassifier ([d1c4eac](https://github.com/npmanos/discourse-labeler/commit/d1c4eac008dd9316aae275c1c294cce50af45173))
* initial project setup, git-flow guidelines, and draft spec ([03089c6](https://github.com/npmanos/discourse-labeler/commit/03089c63991ec9e29a508685d463953e9dc94169))
* initialize Go module and environment configuration ([1479b80](https://github.com/npmanos/discourse-labeler/commit/1479b80f1269a3c4ea9d18a20b1dac5f0b8ed594))
* parse .env file natively and configure custom User-Agent to bypass WAF blocks ([466f33e](https://github.com/npmanos/discourse-labeler/commit/466f33e288f2016a46e1e5b92898fbb86f111d56))
* pass LLM system prompt override option in labeler daemon entrypoint ([1ca89fa](https://github.com/npmanos/discourse-labeler/commit/1ca89fa11ec01cb97b32cbdedaa0dbdfdd0e1a06))
* rename emitted Ozone labels to meta-discourse and possible-meta-discourse ([4eeca24](https://github.com/npmanos/discourse-labeler/commit/4eeca24ed7b87bff71903feddf3b2204e799bdaf))
* track agent collaboration rules in repository ([a0392de](https://github.com/npmanos/discourse-labeler/commit/a0392defdb98f30468714e61c839569a05ce8007))


### Reverts

* Revert "chore: update release-please manifest version to 0.0.0" ([7a158fd](https://github.com/npmanos/discourse-labeler/commit/7a158fd8154936f62a855120ec6105f6e13122f6))



