import ky, { type Options, HTTPError } from 'ky';
import {
	parseError as parseApiError,
	accessTokenExpired,
	refreshTokenExpired
} from '$lib/errors/api';
import { refreshSession } from '$lib/auth';

const baseOptions: Options = {
	retry: {
		limit: 4,
		backoffLimit: 10 * 1000, // 10 seconds,
		shouldRetry: async (s) => {
			if (s.error instanceof HTTPError && (await refreshTokenExpired(s.error.response))) {
				return false;
			}
			return undefined;
		}
	},
	credentials: 'include'
};

const fetch = ky.create({
	...baseOptions,
	hooks: {
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
					await refreshSession(baseOptions);
				} catch (e) {
					if (e instanceof HTTPError) {
						if (await refreshTokenExpired(e.response)) {
							console.error('failed to refresh session:', await e.response.clone().text());
						}
						return e.response;
					}
					throw e;
				}

				return ky.retry();
			}
		]
	}
});

export async function parseError(error: HTTPError): Promise<string> {
	const response = error.response;
	try {
		const parsedError = await parseApiError(response);
		console.error(parsedError);
		return parsedError.message;
	} catch {
		const errorText = await response.text();
		return errorText;
	}
}

type FetchType = typeof fetch;

export type { FetchType };
export { baseOptions };
export default fetch;
