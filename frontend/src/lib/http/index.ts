import ky, { type Options, type KyResponse, HTTPError } from 'ky';
import { ApiError, ApiErrorCodes } from '$lib/errors/api';
import { parseError as parseApiError } from '$lib/errors/api';
import { refreshSession } from '$lib/auth';

const baseOptions: Options = {
	retry: {
		limit: 4,
		backoffLimit: 10 * 1000 // 10 seconds
	},
	credentials: 'include'
};

const isExpiredToken = async (response: KyResponse) => {
	try {
		const res = ApiError.safeParse(await response.json());
		if (!res.success) {
			return false;
		}

		return (
			(res.data.code === ApiErrorCodes.ExpiredToken && response.status === 401) ||
			(res.data.code === ApiErrorCodes.InvalidToken && response.status === 401)
		);
	} catch {
		return false;
	}
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
				const isExpired = await isExpiredToken(response.clone());
				if (!isExpired) {
					return response;
				}

				// Refresh the users session
				try {
					await refreshSession(baseOptions);
				} catch (e) {
					if (e instanceof HTTPError) {
						e.response
							.json()
							.then((val) => {
								console.error(
									`Failed to verify session message=${e.message} body=${JSON.stringify(val)}`
								);
							})
							.catch(() => {
								console.error(`Failed to verify session message=${e.message}`);
							});
					}
					return response;
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
export { baseOptions, isExpiredToken };
export default fetch;
