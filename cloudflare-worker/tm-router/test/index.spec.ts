import { env, createExecutionContext, waitOnExecutionContext, SELF } from 'cloudflare:test';
import { describe, it, expect } from 'vitest';
import worker from '../src';

describe('tm-router worker', () => {
	describe('request for /', () => {
		it('renders the minimal landing page (unit style)', async () => {
			const request = new Request<unknown, IncomingRequestCfProperties>('http://example.com/');
			const ctx = createExecutionContext();
			const response = await worker.fetch(request, env, ctx);
			await waitOnExecutionContext(ctx);
			expect(response.status).toBe(200);
			expect(response.headers.get('Content-Type')).toContain('text/html');

			const text = await response.text();
			expect(text).toContain('Kevin Lu');
			expect(text).toContain('linkedin.com/in/kevin-z-lu');
			expect(text).toContain('/kevin-lu-resume-feb-2026.pdf');
			expect(text).toContain('/tm/replay');
			expect(text).toContain('/foodle');
			expect(text).toContain('/chess_db');
		});

		it('renders the minimal landing page (integration style)', async () => {
			const request = new Request('http://example.com/');
			const response = await SELF.fetch(request);
			expect(response.status).toBe(200);

			const text = await response.text();
			expect(text).toContain('Kevin Lu');
			expect(text).toContain('LinkedIn');
			expect(text).toContain('Resume');
		});
	});

	describe('request for the resume asset', () => {
		it('serves the PDF from static assets (integration style)', async () => {
			const request = new Request('http://example.com/kevin-lu-resume-feb-2026.pdf');
			const response = await SELF.fetch(request);
			expect(response.status).toBe(200);
			expect(response.headers.get('Content-Type')).toContain('application/pdf');

			const signature = new Uint8Array(await response.arrayBuffer()).slice(0, 4);
			expect(Array.from(signature)).toEqual([37, 80, 68, 70]);
		});
	});
});
