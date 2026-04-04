import { afterEach, describe, expect, it, vi } from 'vitest';
import worker from '../src';

afterEach(() => {
	vi.restoreAllMocks();
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
