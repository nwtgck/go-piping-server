# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)

## [Unreleased]

## [0.6.1] - 2023-04-09

## [0.6.0] - 2023-04-09
### Changed
* Update dependencies
* (Docker) golang:1.20

## [0.5.0] - 2023-03-08
### Changed
* Update dependencies
* (Docker) golang:1.19
* Smoother streaming
* Send message to sender

## [0.4.0] - 2022-01-15
### Added
* Support HEAD method for the reserved paths
* Add "Top page" link to /noscript page
* Log Piping Server version
* (Docker) Support linux/arm/v6, linux/arm/v7 and linux/arm64 Docker images

### Changed
* Update dependencies
* Add Go runtime version to --version output and version log

## [0.3.0] - 2021-11-20
### Added
* Make server logging better
* Support HTTP/2 Cleartext (h2c)
* Support X-Piping header passing arbitrary data from sender to receivers
* Create /noscript Web UI for transferring a file without JavaScript
* Support multipart upload

### Changed
* Use requested protocol in /help
* Add `X-Robots-Tag: "none"` header to receiver's response
* Reject POST and PUT with Content-Range for now to detect resumable upload in the future
* Respond 405 Method Not Allowed when method is not supported
* Reject Service Worker registration request

### Fixed
* Not allow receiver do GET a path while transferring
* Add `Access-Control-Allow-Origin: *` to sender's response

## [0.2.0] - 2021-09-18
### Added
* Support preflight request

## 0.1.0 - 2021-05-18
### Added
* Initial release

[Unreleased]: https://github.com/nwtgck/go-piping-server/compare/v0.6.1...HEAD
[0.6.1]: https://github.com/nwtgck/go-piping-server/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/nwtgck/go-piping-server/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/nwtgck/go-piping-server/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/nwtgck/go-piping-server/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/nwtgck/go-piping-server/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/nwtgck/go-piping-server/compare/v0.1.0...v0.2.0
