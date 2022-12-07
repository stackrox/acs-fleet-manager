# Changelog

This Changelog should be updated for:

- Changes in one of the APIs (public, private and admin API)
- Changes in how to operate fleet-manager or fleetshard-sync (e.g new required config values, secrets)
- Changes in the development process (e.g additional required configuration for the e2e test script)

## [NEXT RELEASE]
### Added
### Changed
### Deprecated
### Removed

## 2022-12-06.1.1df0bc5
### Added
- Write cluster params to Parameter Store on cluster IDP setup
- Provision RDS instances and clusters
- Add quay user token when installing Operator from the upstream
### Changed
- Fallback to EVAL if quota check fails
- Switch auth type to RHSSO by default for fleetshard-sync
- Switch auth type to RHSSO by default for fleetshard-sync
### Removed
- Rollback automatic tag resolution on prod

## 2022-11-08.1.3060ea1
### Added
- Data Plane terraforming scripts migration from BitWarden to Parameter Store
- Update go version to 1.18
### Changed
### Deprecated
### Removed
