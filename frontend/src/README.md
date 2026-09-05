# Frontend Structure

React/Vite code is organized by responsibility:

- `app` - application root, routing composition, global styles.
- `pages` - route-level screens.
- `widgets` - large reusable page blocks, such as layout shell.
- `features` - user actions and workflows, such as post creation.
- `entities` - domain types, helpers, and UI for business entities.
- `shared` - infrastructure and reusable primitives without domain ownership.

Legacy server-rendered Jinja templates and static files live outside `src` and are not used as the source of truth for the React app.

## Standalone development

The `frontend` directory is an independent Vite application. It does not import
Python modules or backend files. Configure the API origin with
`VITE_API_BASE_URL` in a local `.env` file; when the variable is absent, the app
uses the current hostname on port `8000` (the Go gateway). Authentication uses
Bearer access/refresh tokens in browser local storage; SSE and WebSocket clients
pass the access token in their query because browser constructors cannot set an
Authorization header.

```bash
npm install
npm run dev
```

The backend only needs to provide the API and allow requests from the frontend
origin. This makes it possible to move the whole `frontend` directory into a
separate repository later without changing the UI code.
