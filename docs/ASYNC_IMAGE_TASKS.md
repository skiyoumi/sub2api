# Asynchronous Image Tasks

Asynchronous image tasks let clients submit long-running OpenAI-compatible image requests without keeping one HTTP connection open. This avoids proxy/CDN response timeouts such as Cloudflare 524 while preserving the existing image routing, billing, moderation, concurrency, and failover behavior.

## Endpoints

The authenticated gateway exposes both `/v1` paths and their existing no-prefix aliases:

```text
POST /v1/images/generations/async
POST /v1/images/edits/async
GET  /v1/images/tasks/{task_id}
GET  /v1/images/tasks
GET  /v1/images/assets/{asset_id}
```

The aliases are `/images/generations/async`, `/images/edits/async`, and `/images/tasks/{task_id}`.

OpenAI and Grok groups, including composite groups routing to these platforms, are supported. Requests use the same JSON or multipart payload as the corresponding synchronous endpoint. Streaming image requests are rejected because a polled task returns one final JSON result.

## Enabling the feature (object storage)

Asynchronous image tasks **use local server storage by default** when cloud credentials are incomplete or absent. Empty legacy S3 defaults migrate to local storage automatically. Explicitly disabling local storage or a configured cloud binding still stops new submissions with `404`. Existing tasks and images remain readable until expiry. This is deliberate: without offloading, large `b64_json` results (several MB each, e.g. `gpt-image-1`) would accumulate in Redis and exhaust its memory.

### From the admin UI (recommended)

**Admin → Backup → Image storage.** Choose local storage, Qiniu Kodo, or a generic S3-compatible store. Saving takes effect immediately. See [Image Studio configuration](image-studio.md) for local volumes and Qiniu settings. The connection test verifies writing, reading and deleting a temporary object.

For S3 storage, the form supports **reusing the backup S3 configuration**: it borrows the endpoint, region and credentials already configured above and keeps only its own bucket and prefix, so backups stay under `backups/` while images go to `images/`. Leave the bucket empty to use the backup bucket as well. Untick the box to point images at a completely separate account.

Saving requires step-up 2FA when that gate is enabled, for the same reason the backup S3 form does: changing the target redirects generated content to another account.

Turning the switch off stops new submissions but keeps already-accepted tasks pollable, so nothing in flight is stranded.

### From the config file

The admin setting takes precedence. When nothing has ever been saved there, the `image_storage` block in `config.yaml` is used instead, so deployments that enabled the feature before the admin UI existed keep using that configuration. Cloud storage additionally requires a persistent encryption key for the deletion journal.

Configure an S3-compatible object store (AWS S3, Cloudflare R2, Aliyun OSS, MinIO, …) in `config.yaml` (all keys also accept the `IMAGE_STORAGE_*` environment overrides):

```yaml
image_storage:
  enabled: true
  provider: "s3"                  # local / qiniu / s3
  local_directory: "./data/generated-images" # used for local storage
  endpoint: "https://<account_id>.r2.cloudflarestorage.com"  # AWS 官方可留空
  region: "auto"
  bucket: "my-images"
  access_key_id: "..."
  secret_access_key: "..."
  prefix: "images/"
  force_path_style: false          # MinIO/path-style buckets set true
  max_download_bytes: 33554432     # cap when re-hosting an upstream image URL (32MB)
```

When a task completes, each generated image is stored and `data[].url` becomes an authenticated `/v1/images/assets/{asset_id}` path. Fetch it with the same API key that submitted the task. `b64_json` is removed before saving the result to Redis. If storage fails, the task is marked `failed` rather than persisting raw base64. Legacy public URL and presign expiry settings no longer apply to these tasks.

New results may also include `data[].preview_url`, an authenticated asset path for a JPEG preview with a maximum edge of 1024 pixels (quality 80, white background for transparency). It is included only when smaller than the original. The studio fetches this preview first and fetches `url` on demand for original downloads and reference reuse. Both assets share task ownership checks and the two-hour retention policy. Original bytes and dimension metadata are unchanged. Old results or images whose preview cannot be created/stored omit this optional field; clients fall back to `url`. Preview decoding is limited to 32 × 1024 × 1024 pixels and two concurrent decodes per process. No schema migration or backfill is needed.

Images expire two hours after storage succeeds. Access is denied at expiry; a worker runs on startup and every minute to delete expired files/objects. PostgreSQL stores a durable deletion journal with the original storage configuration, so cleanup retries survive restarts and configuration changes. Cloud storage requires a persistent `TOTP_ENCRYPTION_KEY`, private buckets and read/write/delete permissions. Existing objects from before this migration are not scanned or deleted.

To add another storage provider, implement both `service.ImageStorage` and `service.ImageBlobStore`, and wrap the provider with `ImageAssetService.Storage` to preserve retention and authenticated access.

### Troubleshooting: the endpoints return 404 after enabling

`404 async image tasks are not enabled` means image creation was explicitly disabled, the local directory is inaccessible, or a configured cloud binding could not be initialized. Missing cloud credentials alone fall back to local storage. The route exists either way — the 404 comes from the handler, not from an unregistered path, which makes it easy to mistake for a missing build.

Check `image_storage.client_build_failed` for local directory/permission problems and `image_storage.settings_load_failed` for settings database errors. Configured cloud storage also requires a persistent encryption key; initialization failures are reported in the backend log.

Note that releases **before v0.1.161 silently dropped `IMAGE_STORAGE_ENDPOINT`, `_BUCKET`, `_ACCESS_KEY_ID`, `_SECRET_ACCESS_KEY` and `_PUBLIC_BASE_URL`** when they were supplied only through the environment: those keys had no registered default, and viper cannot see an environment variable for a key it does not already know about. Deployments driven purely by `environment:` — which is what `deploy/docker-compose.yml` does by default — therefore reported `enabled: true` with empty credentials and 404'd on every async call. On an affected release the workaround is to also place the `image_storage` block in `/app/data/config.yaml` (copy it from `deploy/config.example.yaml`); once the keys exist in the file, the environment overrides apply normally.

Two further causes of a 404 that are unrelated to storage: the API key's group must resolve to the **OpenAI or Grok** platform (including composite routing; unsupported platforms or a key with no group yield `Images API is not supported for this platform`), and a task may only be polled with the **same API key that submitted it** — polling with a different key of the same user returns `image task not found` by design.

## Submit a task

```bash
curl -i https://api.example.com/v1/images/generations/async \
  -H 'Authorization: Bearer sk-...' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-image-1",
    "prompt": "A lighthouse during a winter storm",
    "size": "1536x1024"
  }'
```

The server stores the initial task in Redis and responds with `202 Accepted`:

```json
{
  "id": "imgtask_0123456789abcdef",
  "task_id": "imgtask_0123456789abcdef",
  "object": "image.generation.task",
  "status": "processing",
  "created_at": 1784092800,
  "expires_at": 1784100000,
  "poll_url": "/v1/images/tasks/imgtask_0123456789abcdef"
}
```

`Location` contains the polling path and `Retry-After: 3` provides the recommended polling interval.

Submission, polling and history responses also include `request` with the original `prompt`, `model`, and supplied `size`, `aspect_ratio`, `quality`, and `n`. Only these reusable fields are retained: uploaded references and credentials are excluded. This metadata shares the task record's two-hour expiry and is removed with it. Older tasks may have no `request` field. The studio provides copy/apply actions for the saved prompt; applying it does not submit a new generation.

After offload, `result.data[]` includes `width`, `height`, and `size` probed from the stored PNG/JPEG/WebP bytes. These override upstream claims. A shared `result.size` is present only when every image has the same known dimensions. Unreadable headers do not inherit a claimed size. No resizing occurs. Compare these values with `request.size`; requested dimensions are not a guarantee that the upstream honored them. Quality remains upstream-reported metadata.

## Poll a task

Use the same API key that submitted the task:

```bash
curl https://api.example.com/v1/images/tasks/imgtask_0123456789abcdef \
  -H 'Authorization: Bearer sk-...'
```

While work is in progress:

```json
{
  "id": "imgtask_0123456789abcdef",
  "task_id": "imgtask_0123456789abcdef",
  "object": "image.generation.task",
  "status": "processing",
  "created_at": 1784092800,
  "expires_at": 1784100000
}
```

On success, `result` mirrors the synchronous image API body, except each image has been offloaded to object storage: `data[].url` points at the stored object and `b64_json` is stripped (so both URL and base64 upstream formats end up as compact stored links):

```json
{
  "id": "imgtask_0123456789abcdef",
  "task_id": "imgtask_0123456789abcdef",
  "object": "image.generation.task",
  "status": "completed",
  "http_status": 200,
  "image_url": "/v1/images/assets/550e8400-e29b-41d4-a716-446655440000",
  "result": {
    "created": 1784092923,
    "data": [{"url": "/v1/images/assets/550e8400-e29b-41d4-a716-446655440000"}]
  },
  "created_at": 1784092800,
  "completed_at": 1784092923,
  "expires_at": 1784100123
}
```

For URL responses, `image_url` mirrors the first `data[].url` for simple clients. On failure, the task reaches `failed` and exposes the original OpenAI-compatible error object where available:

```json
{
  "id": "imgtask_0123456789abcdef",
  "task_id": "imgtask_0123456789abcdef",
  "object": "image.generation.task",
  "status": "failed",
  "http_status": 502,
  "error": {
    "type": "api_error",
    "message": "Upstream request failed"
  },
  "created_at": 1784092800,
  "completed_at": 1784092923,
  "expires_at": 1784100123
}
```

All submit and poll responses include `Cache-Control: no-store`, preventing a CDN from caching the `processing` state. Tasks and results expire two hours after their latest state update. A task executes for at most 30 minutes.

Task ownership is scoped to both user and API key. Unknown task IDs and IDs owned by another key both return `404`, avoiding task-existence disclosure. Polling remains available when the completed generation used the key's remaining balance; normal authentication, disabled-key, user, IP, and group checks still apply.
