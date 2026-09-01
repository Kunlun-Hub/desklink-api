# DeskLink client 1.4.9 compatibility

This service is tested against the local DeskLink/RustDesk client source at version `1.4.9`.

## Supported contract

| Client workflow | API endpoints | Compatibility notes |
| --- | --- | --- |
| Password and OIDC login | `/api/login-options`, `/api/login`, `/api/oidc/*`, `/api/logout` | Uses the 1.4.9 token and user payload shape. |
| Current user | `/api/currentUser`, `/api/user/info` | Returns `display_name`, `avatar`, and client status values `1`, `0`, or `-1`. |
| Device heartbeat | `/api/heartbeat`, `/api/sysinfo`, `/api/sysinfo_ver` | Accepts 1.4.9 device and version fields. |
| Accessible devices | `/api/users`, `/api/peers`, `/api/device-group/accessible` | Supports 1.4.9 `current/pageSize` pagination and legacy `page/pageSize`. |
| Legacy address book | `/api/ab` | Preserves the JSON-in-`data` payload used by the client. |
| Personal/shared address books | `/api/ab/settings`, `/api/ab/personal`, `/api/ab/shared/profiles`, `/api/ab/peers`, `/api/ab/tags/*`, `/api/ab/peer/*`, `/api/ab/tag/*` | Uses stable array responses and `current/pageSize` pagination. |
| Web client configuration | `/api/server-config`, `/api/server-config-v2`, `/api/shared-peer` | Available when the Web client is enabled. |
| Audit upload | `/api/audit/conn`, `/api/audit/file` | Accepts 1.4.9 connection and file audit events. |

Run the live contract check against a configured service:

```bash
DESKLINK_TEST_PASSWORD='your-password' scripts/check-client-149.sh
```

The script does not print or persist the password or returned access token.

## hbbs-owned feature

Client 1.4.9 introduces signed `switch-grant` registration for switching control sides while preserving ACL checks. The grant is consumed by a matching 1.4.9 `hbbs` implementation and must be verified against the device public key. This standalone API service does not accept unverified grants. Deploy the corresponding `hbbs` support before enabling that workflow; returning an unconditional success from the API would bypass the security property of the feature.

Session-recording upload is not advertised by this service. In the referenced 1.4.9 client source the upload flag remains disabled by default, so `/api/record` is not part of the active community contract.
