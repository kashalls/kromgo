# Changelog

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
