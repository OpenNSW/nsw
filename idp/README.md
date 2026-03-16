# Identity Provider (IdP) Setup

## Overview

We selected [Thunder](https://github.com/asgardeo/thunder) as the Identity Provider for this project. Thunder is a lightweight, developer-friendly identity and access management solution.

## Getting Started

### Quick Start (with defaults)

Start the IdP server with default credentials (admin/1234):

```bash
docker compose up
```

### Custom Configuration (optional)

To customize admin credentials or other settings:

1. Copy the example environment file:

```bash
cp .env.example .env
```

2. Edit `.env` with your desired values:

```bash
THUNDER_ADMIN_USERNAME=admin
THUNDER_ADMIN_PASSWORD=your-secure-password
```

3. Start the IdP server:

```bash
docker compose up
```

### Developer Console Access

Once the services are running, access the Thunder developer console at `https://localhost:8090/develop`:

- **Default credentials**: admin/1234
- **Custom credentials**: Use the values from your `.env` file

> ⚠️ **Security Warning**: The default password should be changed immediately for non-development environments. Always use strong, unique passwords in production or shared environments.

## Bootstrap Scripts

The docker-compose setup mounts two bootstrap scripts into the Thunder container:

```yaml
- ./01-default-resources.sh:/opt/thunder/bootstrap/01-default-resources.sh
- ./02-sample-resources.sh:/opt/thunder/bootstrap/02-sample-resources.sh
```

These scripts automatically configure Thunder on first startup:

- **`01-default-resources.sh`**: Creates the default organization unit, user schema (Person), admin user, system resource server, admin role, and default authentication/registration flows
- **`02-sample-resources.sh`**: Sets up sample resources including:
  - **Private Sector Organization Unit** - root OU for private-sector entities
  - **ABCD Traders Organization Unit** - child OU under Private Sector
  - **Private_User Type** - user schema for ABCD Traders users
  - **Government Organization Unit** - root OU for government entities
  - **NPQS / FCAU / IRD Organization Units** - child OUs under Government Organization
  - **Government_User Type** - shared user schema for government users
  - **Groups** - `Traders` and `CHA`
  - **Roles** - `Trader` and `CHA` (assigned to matching groups)
  - **Sample Users** - three private users in ABCD Traders and one user per government child OU
  - **SPA Applications** - `TraderApp`, `NPQSPortalApp`, `FCAUPortalApp`, `IRDPortalApp`

## Current Setup

The following resources are configured by bootstrap:

- ✅ Default organization unit and default system resources
- ✅ Private Sector organization unit
- ✅ ABCD Traders child organization unit
- ✅ Government Organization root unit with NPQS, FCAU, and IRD child units
- ✅ Private_User and Government_User user types (schemas)
- ✅ Traders and CHA groups
- ✅ Trader and CHA roles assigned to corresponding groups
- ✅ Three sample private users in ABCD Traders OU with group-based role inheritance
- ✅ One government user in each of NPQS, FCAU, and IRD OUs
- ✅ Four SPA apps with client IDs: `TRADER_PORTAL_APP`, `OGA_PORTAL_APP_NPQS`, `OGA_PORTAL_APP_FCAU`, `OGA_PORTAL_APP_IRD`

## Notes

- Role assignment is **group-based** in sample setup:
  - `Traders` group receives `Trader` role
  - `CHA` group receives `CHA` role
  - Users inherit effective roles from group membership
- Port and app mapping in sample setup:
  - `TraderApp` -> `http://localhost:5173` (`TRADER_PORTAL_APP`)
  - `NPQSPortalApp` -> `http://localhost:5174` (`OGA_PORTAL_APP_NPQS`)
  - `FCAUPortalApp` -> `http://localhost:5175` (`OGA_PORTAL_APP_FCAU`)
  - `IRDPortalApp` -> `http://localhost:5176` (`OGA_PORTAL_APP_IRD`)
- All data is persisted in the `thunder-db` Docker volume
