const HOMEPAGE_ORIGIN = 'https://lukev.github.io/website';

function rewriteTmAssets(response: Response): Response {
	if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('text/html')) {
		return response;
	}
	const rewriteAttribute = (attribute: string) => ({
		element(element: Element) {
			const value = element.getAttribute(attribute);
			if (value?.startsWith('/') && !value.startsWith('//') && !value.startsWith('/tm/')) {
				element.setAttribute(attribute, `/tm${value}`);
			}
		},
	});
	const headers = new Headers(response.headers);
	// Validators and length describe the upstream HTML, not the rewritten body.
	headers.delete('ETag');
	headers.delete('Last-Modified');
	headers.delete('Content-Length');
	headers.set('Cache-Control', 'no-store');
	return new HTMLRewriter()
		.on('script[src]', rewriteAttribute('src'))
		.on('link[href]', rewriteAttribute('href'))
		.transform(new Response(response.body, { status: response.status, statusText: response.statusText, headers }));
}

async function proxyRequest(
	request: Request,
	targetUrl: string,
	options?: { redirect?: RequestInit['redirect'] },
): Promise<Response> {
	const headers = new Headers(request.headers);
	headers.delete('host');

	const response = await fetch(targetUrl, {
		method: request.method,
		headers,
		body: request.body,
		redirect: options?.redirect,
	});

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers: response.headers,
	});
}

export default {
	async fetch(request: Request): Promise<Response> {
		const url = new URL(request.url);
		const path = url.pathname;

		// Route /tm/* to the TM client (frontend)
		// Strip /tm prefix before forwarding - assets are at /assets/, not /tm/assets/
		if (path === '/tm' || path.startsWith('/tm/')) {
			// Strip /tm prefix. /tm -> /, /tm/replay -> /replay, /tm/assets/... -> /assets/...
			const strippedPath = path === '/tm' ? '/' : path.slice(3);
			// A browser may still have the old, unprefixed HTML cached. Fetch its
			// body rather than forwarding a conditional request that returns 304.
			const frontendRequest = new Request(request);
			if (request.headers.get('Accept')?.includes('text/html')) {
				frontendRequest.headers.delete('If-None-Match');
				frontendRequest.headers.delete('If-Modified-Since');
			}
			return rewriteTmAssets(await proxyRequest(frontendRequest, `https://tm-client-production.up.railway.app${strippedPath}${url.search}`));
		}

		// Route /foodle/* to the Foodle client (frontend)
		// Forward the full /foodle path because Foodle is built with basePath=/foodle.
		if (path.startsWith('/foodle')) {
			return proxyRequest(request, `https://foodle-web-production.up.railway.app${path}${url.search}`);
		}

		// Route /chess_db/* to Chess DB frontend.
		// Forward full path because Chess DB is built with basePath=/chess_db.
		if (path.startsWith('/chess_db')) {
			return proxyRequest(request, `https://web-production-d3e49.up.railway.app${path}${url.search}`);
		}

		// Route /poker_solver/* to the poker solver frontend.
		// Forward full path because the app is built with basePath=/poker_solver.
		if (path.startsWith('/poker_solver')) {
			return proxyRequest(request, `https://proxy-production-311f.up.railway.app${path}${url.search}`);
		}

		// Route /api/* to the TM server (backend)
		if (path.startsWith('/api')) {
			const targetUrl = `https://tm-server-production.up.railway.app${path}${url.search}`;
			console.log(`Forwarding API request to: ${targetUrl}`);

			// Special handling for WebSockets
			if (request.headers.get('Upgrade') === 'websocket') {
				return fetch(targetUrl, request);
			}

			return proxyRequest(request, targetUrl, { redirect: 'manual' });
		}

		// Everything else belongs to the standalone homepage project.
		const homepagePath = path === '/' ? '/' : path;
		return proxyRequest(request, `${HOMEPAGE_ORIGIN}${homepagePath}${url.search}`);
	},
};
