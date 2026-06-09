# Changelog

## [0.14.5](https://github.com/home-operations/kromgo/compare/0.14.4...0.14.5) (2026-06-09)


### Features

* **deps:** update module golang.org/x/image (v0.41.0 → v0.42.0) ([#233](https://github.com/home-operations/kromgo/issues/233)) ([c9e267f](https://github.com/home-operations/kromgo/commit/c9e267fd5ccf6d8b5283780d316aa214311cff43))
* **github-release:** update release helm-unittest/helm-unittest (v1.0.3 → v1.1.1) ([#232](https://github.com/home-operations/kromgo/issues/232)) ([a7d09f0](https://github.com/home-operations/kromgo/commit/a7d09f0460b11844bc266431ff9e41994594e22a))
* **mise:** update tool oxfmt (0.53.0 → 0.54.0) ([#234](https://github.com/home-operations/kromgo/issues/234)) ([f4e93a8](https://github.com/home-operations/kromgo/commit/f4e93a8912d6f0573b103762692c5c81475702ed))


### Bug Fixes

* **deps:** update module go.yaml.in/yaml/v4 (v4.0.0-rc.4 → v4.0.0-rc.5) ([#235](https://github.com/home-operations/kromgo/issues/235)) ([eb4b9ef](https://github.com/home-operations/kromgo/commit/eb4b9efece25a82dab941fbb1483d52389564475))

## [0.14.4](https://github.com/home-operations/kromgo/compare/0.14.3...0.14.4) (2026-06-07)


### Features

* **chart:** digest pinning, generated README + values schema, and helm tests ([#227](https://github.com/home-operations/kromgo/issues/227)) ([d0aa57a](https://github.com/home-operations/kromgo/commit/d0aa57ae5f01314bd695bfb3cb135fda721e0e4c))
* **container:** update image mirror.gcr.io/busybox (1.37.0 → 1.38.0) ([#228](https://github.com/home-operations/kromgo/issues/228)) ([03cf4d7](https://github.com/home-operations/kromgo/commit/03cf4d7a83ad914bf1c627f02f3fa9c84ebbf647))
* **deps:** update dependency simple-icons (16.22.0 → 16.23.0) ([#226](https://github.com/home-operations/kromgo/issues/226)) ([11cd211](https://github.com/home-operations/kromgo/commit/11cd21134674d152c35bd9f6c0241e927473be21))


### Bug Fixes

* **chart:** pin the helm-test image as tag@digest so renovate updates both ([#230](https://github.com/home-operations/kromgo/issues/230)) ([ba91657](https://github.com/home-operations/kromgo/commit/ba916575d839d149bd4b5271b259e441e787e6c4))


### Miscellaneous Chores

* remove automerge setting from toolchain groups ([be05683](https://github.com/home-operations/kromgo/commit/be05683d104870c3433669f4b9ef219a7e64cfe2))
* Update release-please-config.json to remove paths ([cce6ef8](https://github.com/home-operations/kromgo/commit/cce6ef8f3c4a0cc1096b38f7ec53defdc50de9ef))

## [0.14.3](https://github.com/home-operations/kromgo/compare/0.14.2...0.14.3) (2026-06-05)


### Features

* **chart:** disable ServiceAccount token automount by default ([#220](https://github.com/home-operations/kromgo/issues/220)) ([2fcdd05](https://github.com/home-operations/kromgo/commit/2fcdd054ddff173b77136c4614266a88afb9df6e))
* **chart:** generate values.schema.json ([#224](https://github.com/home-operations/kromgo/issues/224)) ([f2b1ac9](https://github.com/home-operations/kromgo/commit/f2b1ac9da00ab68bee4fefc34dcdace4ab2e9d35))
* **chart:** render values through tpl ([#218](https://github.com/home-operations/kromgo/issues/218)) ([c04ee32](https://github.com/home-operations/kromgo/commit/c04ee3279f32ea1d4ecf15d144e07172274ddd1d))
* **graph:** optional area fill (fill: true) ([#222](https://github.com/home-operations/kromgo/issues/222)) ([c7045f2](https://github.com/home-operations/kromgo/commit/c7045f2b8e0a01c5f7b945953c9cc61b2fa503af))
* **graph:** y-axis min/max + reference mark lines ([#223](https://github.com/home-operations/kromgo/issues/223)) ([b304342](https://github.com/home-operations/kromgo/commit/b304342603941583f4b64635cdccc7d1372074ca))


### Bug Fixes

* **graph:** round y-axis tick values (PreferNiceIntervals) ([#221](https://github.com/home-operations/kromgo/issues/221)) ([67fcecc](https://github.com/home-operations/kromgo/commit/67fceccf5d59a6ae42108d2d31007fc8ffd638fc))

## [0.14.2](https://github.com/home-operations/kromgo/compare/0.14.1...0.14.2) (2026-06-05)


### Features

* graph valueExpr for y-axis label formatting ([#217](https://github.com/home-operations/kromgo/issues/217)) ([e31db54](https://github.com/home-operations/kromgo/commit/e31db54cb93352fe7fd9941f2dac5654e0d3ff4d))


### Bug Fixes

* no title badge rendering ([#215](https://github.com/home-operations/kromgo/issues/215)) ([e8997f3](https://github.com/home-operations/kromgo/commit/e8997f33f417f9449b473ee1d6a10a5d845e27bf))

## [0.14.1](https://github.com/home-operations/kromgo/compare/0.14.0...0.14.1) (2026-06-05)


### Features

* add kromgo helm chart ([#214](https://github.com/home-operations/kromgo/issues/214)) ([168e479](https://github.com/home-operations/kromgo/commit/168e4791702371cfb37dcdbc0bff8157b34c3641))


### Bug Fixes

* **deps:** update dependency marked (18.0.4 → 18.0.5) ([#213](https://github.com/home-operations/kromgo/issues/213)) ([f1e127d](https://github.com/home-operations/kromgo/commit/f1e127d394925f9f407418a6ae237e98d2d298de))
* **deps:** update module github.com/prometheus/common (v0.68.0 → v0.68.1) ([#210](https://github.com/home-operations/kromgo/issues/210)) ([5f7a860](https://github.com/home-operations/kromgo/commit/5f7a8606283cc4d1074fa1d3e85dff36cbac9538))
* **mise:** update tool go (1.26.3 → 1.26.4) ([cde60a0](https://github.com/home-operations/kromgo/commit/cde60a017bf76153dfef04525520fd851924a38f))

## [0.14.0](https://github.com/home-operations/kromgo/compare/0.13.1...0.14.0) (2026-06-02)


### ⚠ BREAKING CHANGES

* **badge:** unique ids, error badges, labelColor, hand-rolled formatters ([#207](https://github.com/home-operations/kromgo/issues/207))

### Features

* **badge:** unique ids, error badges, labelColor, hand-rolled formatters ([#207](https://github.com/home-operations/kromgo/issues/207)) ([6b69e8c](https://github.com/home-operations/kromgo/commit/6b69e8ccc1bd0f5aa24c51e6800d179d0cc51b4c))

## [0.13.1](https://github.com/home-operations/kromgo/compare/0.13.0...0.13.1) (2026-06-02)


### Features

* **badge:** adapt text color to background + add aria-label/title ([#205](https://github.com/home-operations/kromgo/issues/205)) ([e47ba5a](https://github.com/home-operations/kromgo/commit/e47ba5aaa8dcbacbaff7c0eb8c9259f2be55dc44))

## [0.13.0](https://github.com/home-operations/kromgo/compare/0.12.2...0.13.0) (2026-06-02)


### ⚠ BREAKING CHANGES

* **cache:** add a global cache config block for Cache-Control headers ([#203](https://github.com/home-operations/kromgo/issues/203))

### Features

* **cache:** add a global cache config block for Cache-Control headers ([#203](https://github.com/home-operations/kromgo/issues/203)) ([3175d6f](https://github.com/home-operations/kromgo/commit/3175d6f80f6df643de396b3422f2c771432e3473))
* embed DejaVu Sans + Comic Neue via npm (DejaVu default, shields.io look) ([#202](https://github.com/home-operations/kromgo/issues/202)) ([e52358e](https://github.com/home-operations/kromgo/commit/e52358e5a301224846b9b7f81b801a7becb82c91))

## [0.12.2](https://github.com/home-operations/kromgo/compare/0.12.1...0.12.2) (2026-06-01)


### Features

* stamp build version into the container image ([#201](https://github.com/home-operations/kromgo/issues/201)) ([e1d5bd5](https://github.com/home-operations/kromgo/commit/e1d5bd5a90611f4115a78f47c7033e92f5b06dea))
* support Simple Icons for badge icons alongside MDI ([#199](https://github.com/home-operations/kromgo/issues/199)) ([dc15e13](https://github.com/home-operations/kromgo/commit/dc15e138911b2e188dee14cc21906267313fa442))

## [0.12.1](https://github.com/home-operations/kromgo/compare/0.12.0...0.12.1) (2026-06-01)


### Features

* **mise:** update tool oxfmt (0.52.0 → 0.53.0) ([6cf9b97](https://github.com/home-operations/kromgo/commit/6cf9b978bd742ca97bad35278c3cf7aae89368bb))


### Miscellaneous Chores

* update mise lockfile ([efe5420](https://github.com/home-operations/kromgo/commit/efe542010b36222105fd0edc66d04650d12f6ec8))


### Code Refactoring

* request-scoped logging and comprehensive test cleanup ([#197](https://github.com/home-operations/kromgo/issues/197)) ([eb12795](https://github.com/home-operations/kromgo/commit/eb12795cc1d2b998aa4f13d83188c6eb33c87b18))

## [0.12.0](https://github.com/home-operations/kromgo/compare/0.11.1...0.12.0) (2026-06-01)


### ⚠ BREAKING CHANGES

* typed badge/graph endpoints with themed SVG/PNG graphs (0.12) ([#194](https://github.com/home-operations/kromgo/issues/194))

### Features

* typed badge/graph endpoints with themed SVG/PNG graphs (0.12) ([#194](https://github.com/home-operations/kromgo/issues/194)) ([3c15810](https://github.com/home-operations/kromgo/commit/3c15810905432056bad7abb530246a931e0dfae1))

## [0.11.1](https://github.com/home-operations/kromgo/compare/0.11.0...0.11.1) (2026-06-01)


### Code Refactoring

* reorganize kromgo package and clean up tests ([#192](https://github.com/home-operations/kromgo/issues/192)) ([1e16d0a](https://github.com/home-operations/kromgo/commit/1e16d0a2fdae4ef271e7070fe1cbbdba55bab56d))

## [0.11.0](https://github.com/home-operations/kromgo/compare/v0.10.0...0.11.0) (2026-06-01)


### ⚠ BREAKING CHANGES

* modernize kromgo — CEL config, range queries, caching, lighter deps ([#189](https://github.com/home-operations/kromgo/issues/189))

### Features

* **deps:** update module golang.org/x/image (v0.38.0 → v0.41.0) ([#190](https://github.com/home-operations/kromgo/issues/190)) ([9dca5d4](https://github.com/home-operations/kromgo/commit/9dca5d467256cb87abb45b5075f32ea85daa7adf))
* migrate kromgo to home-operations ([#187](https://github.com/home-operations/kromgo/issues/187)) ([781d872](https://github.com/home-operations/kromgo/commit/781d8724e89681747e355133b7efa3a4dfc8bf70))
* modernize kromgo — CEL config, range queries, caching, lighter deps ([#189](https://github.com/home-operations/kromgo/issues/189)) ([4a227b1](https://github.com/home-operations/kromgo/commit/4a227b1dd419e2db22ec5dc001fdf1b3408e806e))


### Bug Fixes

* **deps:** update module github.com/go-chi/chi/v5 to v5.3.0 ([#184](https://github.com/home-operations/kromgo/issues/184)) ([282a24e](https://github.com/home-operations/kromgo/commit/282a24e5b9235d5d93d59e6a23b3ff58d305b99c))
* **deps:** update module github.com/prometheus/common to v0.68.0 ([#185](https://github.com/home-operations/kromgo/issues/185)) ([3f95ab2](https://github.com/home-operations/kromgo/commit/3f95ab224c8a8eb0710ec24b87691dd3dba3c1ee))

## Changelog
