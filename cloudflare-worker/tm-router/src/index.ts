const landingPageHtml = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Kevin Lu</title>
  <style>
    :root {
      color-scheme: light;
      --bg-start: #fbfbf8;
      --bg-end: #f1f0eb;
      --surface: rgba(255, 255, 255, 0.86);
      --text: #171717;
      --muted: #666660;
      --line: #dddcd4;
      --link: #1f4b99;
      --shadow: rgba(18, 18, 18, 0.06);
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      min-height: 100vh;
      padding: 32px 20px;
      background: linear-gradient(180deg, var(--bg-start) 0%, var(--bg-end) 100%);
      color: var(--text);
      font-family: Georgia, "Times New Roman", serif;
    }

    main {
      max-width: 720px;
      margin: 0 auto;
      padding: 40px 32px;
      border: 1px solid var(--line);
      border-radius: 24px;
      background: var(--surface);
      box-shadow: 0 20px 48px var(--shadow);
      backdrop-filter: blur(8px);
    }

    h1 {
      margin: 0 0 10px;
      font-size: clamp(2.2rem, 8vw, 3.75rem);
      font-weight: 600;
      letter-spacing: -0.05em;
    }

    p {
      margin: 0;
      color: var(--muted);
      font-size: 1.05rem;
      line-height: 1.7;
    }

    section {
      margin-top: 32px;
      padding-top: 18px;
      border-top: 1px solid var(--line);
    }

    h2 {
      margin: 0 0 14px;
      color: var(--muted);
      font-family: "SFMono-Regular", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.75rem;
      font-weight: 600;
      letter-spacing: 0.18em;
      text-transform: uppercase;
    }

    .links {
      display: flex;
      flex-wrap: wrap;
      gap: 16px;
    }

    .links a {
      color: var(--link);
    }

    ul {
      margin: 0;
      padding: 0;
      list-style: none;
    }

    li {
      display: flex;
      justify-content: space-between;
      gap: 20px;
      align-items: baseline;
      padding: 11px 0;
      border-bottom: 1px solid #ecebe5;
    }

    li:last-child {
      border-bottom: none;
      padding-bottom: 0;
    }

    a {
      color: var(--text);
      text-decoration: none;
      border-bottom: 1px solid transparent;
    }

    a:hover {
      border-color: currentColor;
    }

    .meta {
      color: var(--muted);
      font-size: 0.95rem;
      text-align: right;
    }

    @media (max-width: 640px) {
      main {
        padding: 28px 22px;
        border-radius: 18px;
      }

      li {
        display: block;
      }

      .meta {
        display: block;
        margin-top: 4px;
        text-align: left;
      }
    }
  </style>
</head>
<body>
  <main>
    <h1>Kevin Lu</h1>
    <p>Software engineer.</p>

    <section>
      <h2>Links</h2>
      <div class="links">
        <a href="https://linkedin.com/in/kevin-z-lu" target="_blank" rel="noreferrer">LinkedIn</a>
        <a href="https://github.com/lukev" target="_blank" rel="noreferrer">GitHub</a>
      </div>
    </section>

    <section>
      <h2>Projects</h2>
      <ul>
        <li>
          <a href="/tm">Terra Mystica Server</a>
          <span class="meta">live games and replay tools</span>
        </li>
        <li>
          <a href="/foodle">Foodle</a>
          <span class="meta">daily food word game</span>
        </li>
        <li>
          <a href="/chess_db">Chess DB</a>
          <span class="meta">searchable chess database</span>
        </li>
        <li>
          <a href="/poker_solver">Poker Solver</a>
          <span class="meta">browser UI for solver workflows</span>
        </li>
      </ul>
    </section>
  </main>
</body>
</html>`;

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

		// Route /foodle/* to the Foodle client (frontend)
		// Forward the full /foodle path because Foodle is built with basePath=/foodle.
		if (path.startsWith('/foodle')) {
			const targetUrl = `https://foodle-web-production.up.railway.app${path}${url.search}`;

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

		// Route /chess_db/* to Chess DB frontend.
		// Forward full path because Chess DB is built with basePath=/chess_db.
		if (path.startsWith('/chess_db')) {
			const targetUrl = `https://web-production-d3e49.up.railway.app${path}${url.search}`;

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

		// Route /poker_solver/* to the poker solver frontend.
		// Forward full path because the app is built with basePath=/poker_solver.
		if (path.startsWith('/poker_solver')) {
			const targetUrl = `https://proxy-production-311f.up.railway.app${path}${url.search}`;
			const headers = new Headers(request.headers);
			headers.delete('host');

			const response = await fetch(targetUrl, {
				method: request.method,
				headers,
				body: request.body,
			});

			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers,
			});
		}

		// Route /api/* to the TM server (backend)
		if (path.startsWith('/api')) {
			const targetUrl = `https://tm-server-production.up.railway.app${path}${url.search}`;
			console.log(`Forwarding API request to: ${targetUrl}`);

			// Special handling for WebSockets
			if (request.headers.get('Upgrade') === 'websocket') {
				return fetch(targetUrl, request);
			}

			const response = await fetch(targetUrl, {
				method: request.method,
				headers: request.headers,
				body: request.body,
				redirect: 'manual', // Don't follow redirects, let the browser handle them
			});

			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers,
			});
		}

		if (path === '/' || path === '/index.html') {
			return new Response(landingPageHtml, {
				headers: { 'Content-Type': 'text/html; charset=UTF-8' },
			});
		}

		return new Response('Not found', { status: 404 });
	},
};
