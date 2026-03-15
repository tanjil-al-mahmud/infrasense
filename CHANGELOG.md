# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Production QA Release

### Added
- Complete automated end-to-end device monitoring platform via Docker Compose stack.
- Redfish, IPMI, SNMP, Proxmox, and SSH metrics and logs collectors.
- TypeScript React dashboard with device management, alert rules, and RBAC authentication.
- Real-time device stream viewing capability.
- Full Windows Native deployment support and native Linux support.
- API validation and comprehensive security middleware.
- Graceful shutdown protocols and robust error handling across all backend services.
- Detailed, automated deployment and health checks workflows for Windows and Linux.
- Extensive documentation for configurations, upgrades, troubleshooting, and API endpoints.

### Changed
- Hardened all services inside Docker Compose to run with specific memory limits to enhance resource isolation.
- Implemented 30-second bounded timeouts and JSON structured logging with X-Request-ID propagation.
- Re-architected Reverse Proxy (Nginx) to include security headers (X-Frame-Options, X-XSS-Protection), gzip compression, and client payload constraints.
- Optimized frontend device registration form workflow with robust Protocol Auto-Detection sequences.

### Fixed
- Fixed TypeScript `DeviceForm.tsx` compile warnings and unreachable protocol order declarations.
- Removed hardcoded fallback administrative credential values from backend initialization scripts in favor of secure configuration overrides.
- Corrected Promtail container missing `/var/run/docker.sock` volume bindings to enable correct log streaming.
- Fixed backend API `routes.go` CORS directives to dynamically inherit from `CORS_ALLOWED_ORIGINS` environment definitions.
- Set Linux shell script proper execute configurations inside Git via `.gitattributes`.

### Removed
- Cleaned and removed temporary test/scratch files spanning `fix_admin.sql`, `genhash`, and `test` suites.
- Removed unnecessary metadata tracking records and redundant project file overhead.
