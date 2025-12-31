import ky, { type KyResponse, type Options, HTTPError } from 'ky';
import { accessTokenExpired, refreshTokenExpired } from '$lib/errors/api';
import { CSRF_HEADER, CSRF_TOKEN_COOKIE_NAME, refreshSession } from '$lib/auth';

const retryCodes = [408, 413, 429, 500, 502, 503, 504];

/**
 * Injects CSRF token into request headers for state-changing requests.
 * Only works in browser context.
 */
function injectCSRFToken(request: Request) {
	if (
		typeof window !== 'undefined' &&
		['POST', 'PUT', 'DELETE', 'PATCH'].includes(request.method.toUpperCase())
	) {
		document.cookie.split(';').map((c) => {
			c = c.trim();
			const splitIdx = c.indexOf('=');
			const key = c.slice(0, splitIdx);
			const val = c.slice(splitIdx + 1);
			if (key === CSRF_TOKEN_COOKIE_NAME) {
				request.headers.set(CSRF_HEADER, val);
			}
		});
	}
}

const baseOptions: Options = {
	timeout: 15 * 1000,
	retry: {
		retryOnTimeout: true,
		limit: 4,
		backoffLimit: 10 * 1000, // 10 seconds,
		shouldRetry: async (s) => {
			if (s.error instanceof HTTPError && (await refreshTokenExpired(s.error.response))) {
				return false;
			}
			return undefined;
		},
		statusCodes: retryCodes
	},
	credentials: 'include',
	hooks: {
		beforeRequest: [
			(request) => {
				injectCSRFToken(request);
			}
		]
	}
};

const fetch = ky.create({
	...baseOptions,
	hooks: {
		...baseOptions?.hooks,
		afterResponse: [
			async (request, options, response) => {
				if (response.ok) {
					return response;
				}

				// Exit if the token is not expired.
				// The request failed for another reason.
				const isExpired = await accessTokenExpired(response);
				if (!isExpired) {
					return response;
				}

				// Refresh the users session
				try {
					await refreshSession({}, baseOptions);
				} catch (e) {
					if (e instanceof HTTPError) {
						if (await refreshTokenExpired(e.response)) {
							console.error('failed to refresh session:', await e.response.clone().text());
						}
						return e.response;
					}
					throw e;
				}

				// Inject the new CSRF token for the retry
				injectCSRFToken(request);

				// Retry the original request with the new access token and CSRF token
				return ky(request);
			}
		]
	}
});

export function isRetryable(response: KyResponse) {
	return retryCodes.includes(response.status);
}

type FetchType = typeof fetch;

export type { FetchType };
export { baseOptions };
export default fetch;
