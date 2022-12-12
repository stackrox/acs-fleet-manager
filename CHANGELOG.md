# Changelog

This Changelog should be updated for:

- Changes in one of the APIs (public, private and admin API)
- Changes in how to operate fleet-manager or fleetshard-sync (e.g new required config values, secrets)
- Changes in the development process (e.g additional required configuration for the e2e test script)

## [NEXT RELEASE]
### Added
### Changed
- Data Plane terraforming now deploys fleetshard image obtained dynamically rather than hardcoded in the script
- Upgrade StackRox operator to v3.73.0
- Add managed DB values to the Data Plane terraforming Helm Chart
### Deprecated
### Removed

## 2022-11-08.1.3060ea1
### Added
- Data Plane terraforming scripts migration from BitWarden to Parameter Store
- Update go version to 1.18
### Changed
### Deprecated
### Removed
