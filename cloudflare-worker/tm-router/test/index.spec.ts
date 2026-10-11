import { afterEach, describe, expect, it, vi } from 'vitest';
import worker from '../src';

afterEach(() => {
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

describe('tm-router worker', () => {
	it('proxies the homepage to the standalone website project', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response('homepage', {
				status: 200,
				headers: { 'Content-Type': 'text/html; charset=UTF-8' },
			}),
		);
		vi.stubGlobal('fetch', fetchMock);

		const response = await worker.fetch(new Request('https://kezilu.com/'));

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock).toHaveBeenCalledWith(
			'https://lukev.github.io/website/',
			expect.objectContaining({ method: 'GET' }),
		);
		expect(await response.text()).toBe('homepage');
	});

	it('strips the /tm prefix before proxying to the TM frontend', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response('tm-client'));
		vi.stubGlobal('fetch', fetchMock);

		await worker.fetch(new Request('https://kezilu.com/tm/replay/123?view=full'));

		expect(fetchMock).toHaveBeenCalledWith(
			'https://tm-client-production.up.railway.app/replay/123?view=full',
			expect.objectContaining({ method: 'GET' }),
		);
	});

	it.each(['/tm', '/tm/', '/tm/game/1', '/tm/replay/123'])('rewrites frontend asset URLs at %s without changing other links', async (path) => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(`<!doctype html>
			<script type="module" src="/assets/index.js"></script>
			<link rel="stylesheet" href="/assets/index.css">
			<link rel="icon" href="/vite.svg">
			<link rel="modulepreload" href="/tm/assets/chunk.js">
			<script src="https://cdn.example.com/library.js"></script>
			<script src="//cdn.example.com/library.js"></script>
			<script src="./relative.js"></script>
			<a href="/api/replay/export">Export</a>`, {
			status: 200,
			headers: { 'Content-Type': 'text/html; charset=UTF-8', ETag: 'upstream', 'Last-Modified': 'Wed, 01 Oct 2025 12:00:00 GMT', 'Content-Length': '1000' },
		})));

		const response = await worker.fetch(new Request(`https://kezilu.com${path}`));
		const html = await response.text();
		expect(html).toContain('src="/tm/assets/index.js"');
		expect(html).toContain('href="/tm/assets/index.css"');
		expect(html).toContain('href="/tm/vite.svg"');
		expect(html).toContain('href="/tm/assets/chunk.js"');
		expect(html).not.toContain('/tm/tm/');
		expect(html).toContain('src="https://cdn.example.com/library.js"');
		expect(html).toContain('src="//cdn.example.com/library.js"');
		expect(html).toContain('src="./relative.js"');
		expect(html).toContain('href="/api/replay/export"');
		expect(response.headers.get('ETag')).toBeNull();
		expect(response.headers.get('Last-Modified')).toBeNull();
		expect(response.headers.get('Content-Length')).toBeNull();
		expect(response.headers.get('Cache-Control')).toBe('no-store');
	});

	it('fetches full HTML when the browser has the old asset links cached', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response('<html></html>', { headers: { 'Content-Type': 'text/html' } }));
		vi.stubGlobal('fetch', fetchMock);
		await worker.fetch(new Request('https://kezilu.com/tm/', {
			headers: { Accept: 'text/html', 'If-None-Match': 'old-html', 'If-Modified-Since': 'Wed, 01 Oct 2025 12:00:00 GMT' },
		}));
		const headers = fetchMock.mock.calls[0][1].headers as Headers;
		expect(headers.has('If-None-Match')).toBe(false);
		expect(headers.has('If-Modified-Since')).toBe(false);
	});

	it('proxies prefixed assets unchanged and keeps their cache validators', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response('export const content = "/assets/image.png";', {
			headers: { 'Content-Type': 'application/javascript', ETag: 'asset', 'Cache-Control': 'public, max-age=31536000' },
		}));
		vi.stubGlobal('fetch', fetchMock);
		const response = await worker.fetch(new Request('https://kezilu.com/tm/assets/index.js?v=1', { headers: { 'If-None-Match': 'asset' } }));
		expect(fetchMock).toHaveBeenCalledWith('https://tm-client-production.up.railway.app/assets/index.js?v=1', expect.anything());
		expect(fetchMock.mock.calls[0][1].headers.get('If-None-Match')).toBe('asset');
		expect(await response.text()).toBe('export const content = "/assets/image.png";');
		expect(response.headers.get('ETag')).toBe('asset');
		expect(response.headers.get('Cache-Control')).toBe('public, max-age=31536000');
	});

	it('does not claim homepage paths that only start with the letters tm', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response('homepage'));
		vi.stubGlobal('fetch', fetchMock);
		await worker.fetch(new Request('https://kezilu.com/tm-notes'));
		expect(fetchMock).toHaveBeenCalledWith('https://lukev.github.io/website/tm-notes', expect.anything());
	});

	it('proxies API requests with manual redirect handling', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response('api'));
		vi.stubGlobal('fetch', fetchMock);

		await worker.fetch(new Request('https://kezilu.com/api/replay/start', { method: 'POST', body: 'x=1' }));

		expect(fetchMock).toHaveBeenCalledWith(
			'https://tm-server-production.up.railway.app/api/replay/start',
			expect.objectContaining({
				method: 'POST',
				redirect: 'manual',
			}),
		);
	});
});
