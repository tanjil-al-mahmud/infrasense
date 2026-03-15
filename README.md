# InfraSense - Hybrid Infrastructure Monitoring Platform

Monitor all your servers, UPS devices, and virtualization platforms from a single dashboard.

## Quick Start - Windows
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense
copy .env.example .env
cd deploy && docker compose --env-file ..\.env up -d --build
Open: http://localhost | Login: admin / Admin@123456

## Quick Start - Ubuntu/Linux
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense && cp .env.example .env
cd deploy && sudo docker compose --env-file ../.env up -d --build
Open: http://localhost | Login: admin / Admin@123456

## Supported Devices
| Vendor | Protocol | Models |
|--------|----------|--------|
| Dell | Redfish/IPMI | iDRAC7/8/9 |
| HPE | Redfish/IPMI | iLO3/4/5 |
| Supermicro | Redfish/IPMI | All supported |
| Lenovo | Redfish/IPMI | XClarity |
