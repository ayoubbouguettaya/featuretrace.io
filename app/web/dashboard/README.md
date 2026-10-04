# FeatureTrace dashboard

Log viewer for FeatureTrace, built with React and Vite. It reads everything from the query API (`/v1/logs`, `/v1/facets`).

- **Logs**: newest first, filtered by service, feature, level, message text and time range. Filters are kept in the URL, so a view can be shared.
- **Live**: refreshes every 2 seconds and briefly highlights new logs.
- **Trace view** (`?trace_id=…`): every log of one request across services, oldest first, with a waterfall of the spans (one bar per service hop).
- Click a log to see all of its fields, its metadata and its stack trace. Click a service or feature in the table to filter by it.

## Develop

Run the query API on the host (`make run-query` in `app/`), then:

```bash
npm install
npm run dev     # http://localhost:5173, proxies /v1 to localhost:3008
```

Set `QUERY_API_URL` to proxy to another query API.

## Run in Docker

nginx serves the build and proxies `/v1` to `QUERY_API_URL`:

```bash
docker build -t featuretrace/dashboard .
docker run -p 3009:80 -e QUERY_API_URL=http://query-api:3008 featuretrace/dashboard
```
