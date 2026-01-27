export default {
	async fetch(request: Request): Promise<Response> {
		const url = new URL(request.url);
		const path = url.pathname;

		// Route /tm/* to the TM client (frontend)
		// Strip /tm prefix before forwarding - assets are at /assets/, not /tm/assets/
		if (path.startsWith('/tm')) {
			// Strip /tm prefix. /tm -> /, /tm/replay -> /replay, /tm/assets/... -> /assets/...
			const strippedPath = path === '/tm' ? '/' : path.slice(3);
			const targetUrl = `https://tm-client-production.up.railway.app${strippedPath}${url.search}`;

			// Forward the request
			const response = await fetch(targetUrl, {
				method: request.method,
				headers: request.headers,
				body: request.body,
			});

			// Return the response
			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers,
			});
		}

		// Route /api/* to the TM server (backend)
		// Backend expects /api/... paths, so we forward the path as-is
		if (path.startsWith('/api')) {
			const targetUrl = `https://tm-server-production.up.railway.app${path}${url.search}`;

			const response = await fetch(targetUrl, {
				method: request.method,
				headers: request.headers,
				body: request.body,
			});

			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers,
			});
		}

		// Default: return a simple landing page or 404
		return new Response(
			`<!DOCTYPE html>
<html>
<head>
  <title>kezilu.com</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; }
    h1 { color: #333; }
    a { color: #0066cc; }
    ul { line-height: 2; }
  </style>
</head>
<body>
  <h1>Welcome to kezilu.com</h1>
  <p>Available apps:</p>
  <ul>
    <li><a href="/tm/replay">Terra Mystica Log Replayer</a></li>
  </ul>
</body>
</html>`,
			{
				headers: { 'Content-Type': 'text/html' },
			}
		);
	},
};
