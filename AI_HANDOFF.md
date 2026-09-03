# DeskLink API AI Handoff

Last verified: 2026-09-03 UTC.

This repository contains the Go API, React administration UI, Web client assets,
MySQL schema migration, and the production Compose definition for DeskLink. The
product remote is `origin` (`Kunlun-Hub/desklink-api`); `upstream` is the original
community project.

## Current Production Snapshot

- Host: `10.202.22.90`; deployment directory: `/opt/desklink`
- API image: `ohoimager/desklink-api:1.4.9-session-merge-v5`
- Docker Hub manifest digest:
  `sha256:b69bf8af6c8cb10e146f16e2ce451c90f6d0b5a7a2b80c97b2cfc60c88042c66`
- Server image: `ohoimager/desklink-server:1.4.9-secure-tcp`
- Database: MySQL 8.4, schema version `269`
- Containers last verified healthy: `desklink-api`, `desklink-hbbs`,
  `desklink-hbbr`, and `desklink-mysql`
- Current recording storage selection: local `runtime/recordings`
- External mount exposed to the API container: `/recordings-external`, backed
  by host `${DESKLINK_RECORDING_MOUNT_PATH}` (currently
  `/opt/desklink/recordings-external`)

This is a dated snapshot. Verify live state before deploying or reporting it.
Credentials are supplied out of band; never put them in this document or Git.

## Architecture and Ownership

- `cmd/apimain.go`: startup and schema version. Every model requiring migration
  must be included in `Migrate`; bump `DatabaseVersion` deliberately.
- `docker-compose.mysql.yml`: authoritative production topology and image tags.
- `service/recording.go`: recording policy, chunk upload, checksum, completion,
  transcode, retention, and metadata lifecycle.
- `service/recording_storage.go`: local/mounted filesystem, FTP/FTPS, and S3
  object-store adapters plus encrypted storage settings.
- `service/recording_merge.go`: debounced merge of segments sharing a remote
  `peer_id + from_peer + session_id`; display/encoder changes within one live
  session should not remain as separate final recordings.
- `model/recording.go`: recording, policy, and versioned storage setting models.
- `http/controller/api/recording.go`: device upload and signed content delivery.
- `http/controller/admin/recording.go`: admin policy/storage/list endpoints.
- `web/src/pages/RecordingsPage.tsx`: recording policy, external storage form,
  list, preview, download, and delete UI.

Do not move ID/Relay protocol behavior into this repository. That belongs to
`/root/DeskLink Server`. Client heartbeat and recording uploads belong to
`/root/rustdesk`.

## Recording Storage Contract

Supported backends:

- `local`: API filesystem path.
- `nfs` and `smb`: already-mounted filesystem paths. The API container must not
  receive mount privileges. Mount NFS/CIFS on the host and map it into
  `/recordings-external`.
- `ftp`: FTP or explicit TLS depending on `secure`.
- `s3`: S3-compatible service such as MinIO.

Uploads and H.265 preview transcoding always stage under
`RUSTDESK_API_RECORDING_PATH` (`/app/runtime/recordings` in production). On
completion, files are archived to the selected backend. Each recording stores
`storage_setting_id`; changing a bucket/path/backend affects new recordings only
and must not break old preview, download, retention, or deletion.

When a client rebuilds its encoder for display or parameter changes, the API
receives multiple completed segments. A five-second debounce merges segments
with the same `peer_id`, `from_peer`, and non-empty `session_id`. Stream copy is
attempted only when codecs and containers match; mixed codecs or incompatible
streams are normalized to one resolution and re-encoded as H.264 MP4. The
earliest row remains as the audit record and duration/size/hash are recomputed.
Do not merge rows with an empty session ID.

FTP/S3 secrets are AES-GCM encrypted using a key derived from
`RUSTDESK_API_JWT_KEY`. The admin API returns only `has_password` and
`has_secret_key`, never plaintext. Do not rotate the JWT key without a credential
migration or saved external credentials become unreadable. Saving a storage
configuration performs a real write/delete connection test.

## Local Verification

```bash
go test ./service ./http/controller/api ./http/controller/admin ./http/router ./model
cd web
npm run lint
npm run build
cd ..
docker build --pull=false -f Dockerfile.production -t desklink-api:local-test .
```

`go test ./...` currently fails only in pre-existing Redis/cache tests that
hard-code `192.168.1.168:6379`. Do not treat that timeout as a recording change
regression; still inspect any other failure.

Real FTP/S3 integration tests are opt-in:

```bash
DESKLINK_TEST_FTP_ENDPOINT=127.0.0.1:1921 \
DESKLINK_TEST_FTP_USERNAME=desklink \
DESKLINK_TEST_FTP_PASSWORD='<test-only>' \
DESKLINK_TEST_S3_ENDPOINT=127.0.0.1:19000 \
DESKLINK_TEST_S3_ACCESS_KEY='<test-only>' \
DESKLINK_TEST_S3_SECRET_KEY='<test-only>' \
go test ./service -run TestRecordingObjectStoreIntegration -v
```

Use disposable local services and credentials. The test covers backend check,
archive, materialize, and delete. Mounted-filesystem and storage-version
compatibility are covered by `TestMountedRecordingStorageLifecycle`.

The repository intentionally tracks `go.sum` even though legacy `.gitignore`
contains it; production Docker builds copy both `go.mod` and `go.sum`. When
dependencies change, stage it explicitly with `git add -f go.sum`.

## Production Deployment

Before changing production:

1. Verify the requested image and Git commit.
2. Back up `/opt/desklink/.env`, `docker-compose.mysql.yml`, and MySQL.
3. Never print `.env`, tokens, passwords, or private keys to logs/chat.
4. Build and test locally before push/deploy.
5. Push Docker Hub and verify the remote manifest digest.
6. Copy the reviewed Compose file and deploy only the API when server changes
   are not required.

Typical commands, with credentials obtained out of band:

```bash
docker build -f Dockerfile.production \
  -t ohoimager/desklink-api:<tag> .
docker push ohoimager/desklink-api:<tag>
ssh root@10.202.22.90
cd /opt/desklink
docker compose -f docker-compose.mysql.yml up -d --no-deps api
docker inspect --format '{{.Config.Image}} {{.State.Health.Status}}' desklink-api
curl -fsS http://127.0.0.1:21114/api/version
```

Docker Hub has intermittent `EOF`/TLS failures. Do not confuse a registry error
with a failed build or deployment. `skopeo copy --dest-authfile
~/.docker/config.json docker-daemon:<image> docker://docker.io/<image>` has been a
useful fallback. For urgent deployment, `docker save | ssh docker load` is valid,
but the user explicitly requires Docker Hub push too, so finish and verify both.

After startup, verify schema version, container health, admin storage GET/POST,
external mount writability as UID 10001, and an existing recording with a
one-byte Range request. Preview/download should return `206`; download should
also return the original `Content-Disposition` filename.

## Backups and Rollback

Known pre-change backups on the production host include:

- `/opt/desklink/.env.before-external-storage-20260902T1510Z`
- `/opt/desklink/docker-compose.mysql.yml.before-external-storage-20260902T1510Z`
- `/opt/desklink/mysql-before-external-storage-20260902T1510Z.sql`
- `/opt/desklink/mysql-before-recording-20260902T1028Z.sql`

Do not assume they are recent enough for a future rollback. Create a fresh
timestamped backup before every deployment. Restoring an old database is
destructive and requires explicit user authorization.

## Security and Operational Warnings

- Port `21120` is the hbbs internal grant API and must remain loopback-only.
- MySQL must remain bound to `127.0.0.1` unless a reviewed network design says
  otherwise.
- Preserve the `desklink_server_data` volume. Losing it changes the server key
  and invalidates embedded client trust.
- Do not expose raw storage credentials through API responses or UI state.
- Do not delete recording metadata before the backing object has been deleted
  successfully.
- Preserve unrelated worktree changes and do not edit `/root/cloink` for this
  deployment.
