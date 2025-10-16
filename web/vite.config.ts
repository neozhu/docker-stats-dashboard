import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig(() => {
	return {
		plugins: [tailwindcss(), sveltekit()],
		build: {
			rollupOptions: {
				external: ['ws']
			}
		},
		server: {
			// Allow custom host for dev access (added per request)
			allowedHosts: ['docker-stats.blazorserver.com']
		},
		test: {
			globals: true,
			environment: 'jsdom',
			include: ['src/**/*.{test,spec}.{ts,js}'],
			setupFiles: ['./src/setupTests.ts']
		}
	};
});
