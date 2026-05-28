# 🌉 Integration Guide: Building on musu-marketer

`musu-marketer` is designed to be an extensible engine. You can integrate it into your own products via the **Publisher Registry** or the **REST API**.

---

## 1. Extending with Custom Publishers
If you need to push campaigns to your own proprietary CMS, CRM, or a custom Slack channel, you can write a simple Go adapter.

### Step-by-step:
1. Create a new file in `internal/publisher/your_adapter.go`.
2. Implement the `Publisher` interface.
3. Register it in an `init()` function using `publisher.Register()`.
4. Rebuild the project. You can now use `.\musu-marketer.exe publish [ID] --platform your-key`.

---

## 2. Remote Control via REST API (v1.2.1)
Start the server to allow your application to trigger marketing tasks remotely:
```bash
./musu-marketer serve --port 8081
```

### Endpoints:
#### `GET /health`
Returns the status of the engine.
```json
{"status": "alive"}
```

#### `POST /api/v1/draft` (Work in Progress)
Trigger the Strategist and Copywriter pipeline for a specific topic.

#### `GET /api/v1/campaigns` (Work in Progress)
List the drafted and published campaigns from the SQLite database.

---

## 3. Embedding the Engine (Go Developers)
`musu-marketer` is highly modular. You can import its internal packages directly into your Go backend:

- `internal/agent`: Invoke the Strategist or Copywriter agents programmatically.
- `internal/db`: Access the persistent campaign storage.
- `internal/bridge`: Scan and ingest knowledge from any directory.
