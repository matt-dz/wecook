import { verifySession, refreshSession, ACCESS_TOKEN_COOKIE_NAME, type Role } from '$lib/auth';
import ky, { HTTPError, type Options } from 'ky';
import { baseOptions } from '$lib/http';
import { REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import {
	accessTokenExpired,
	ApiErrorCodes,
	parseError,
	refreshTokenExpired
} from '$lib/errors/api';
import { redirect, type Handle, type HandleServerError } from '@sveltejs/kit';
import * as setCookie from 'set-cookie-parser';

export const handle: Handle = async ({ event, resolve }) => {
	const patchCookies = (setCookieHeader: string[]) => {
		setCookie.parse(setCookieHeader).map(({ name, value, ...opts }) => {
			event.cookies.set(name, value, {
				...opts,
				sameSite: opts.sameSite as boolean | 'lax' | 'strict' | 'none' | undefined,
				path: opts.path ?? '/'
			});
		});
	};

	const accessToken = event.cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	const refreshToken = event.cookies.get(REFRESH_TOKEN_COOKIE_NAME);

	// Base case when no auth is provided
	if (!accessToken && !refreshToken) {
		if (event.route.id === '/login') {
			return await resolve(event);
		}
	}

	// Refresh session if necessary
	if (!accessToken && refreshToken) {
		try {
			const res = await refreshSession({
				refresh_token: refreshToken
			});
			patchCookies(res.headers.getSetCookie());
		} catch (e) {
			console.error('failed to refresh session', e);
			if (e instanceof HTTPError && e.response.status === 401) {
				console.error('invalid refresh token');
				event.cookies.delete(REFRESH_TOKEN_COOKIE_NAME, { path: '/' });
				redirect(303, '/login');
			}
			// TODO: render 500 screen
			redirect(303, '/');
		}
	}

	// Exit early, no auth required
	if (!event.route.id?.startsWith('/(user)/') && !event.route.id?.startsWith('/(admin)/')) {
		return await resolve(event);
	}

	// Extract required role
	let role: Role;
	if (event.route.id?.startsWith('/(admin)/')) {
		role = 'admin';
	} else {
		role = 'user';
	}

	const concatenateCookies = (cookies: typeof event.cookies) =>
		cookies
			.getAll()
			.map(({ name, value }) => `${name}=${value}`)
			.join('; ');

	const options: Options = {
		...baseOptions,
		headers: {
			Cookie: concatenateCookies(event.cookies)
		},
		hooks: {
			afterResponse: [
				async (request, options, response) => {
					// Exit if token is not expired.
					// Failed for another reason.
					if (!(await accessTokenExpired(response))) {
						return response;
					}
					console.log('attempting to retry');
					const refreshToken = event.cookies.get(REFRESH_TOKEN_COOKIE_NAME);
					if (!refreshToken) {
						return redirect(303, '/login');
					}

					try {
						// Refresh the session
						console.log('refreshing session');
						const res = await refreshSession(
							{
								refresh_token: refreshToken
							},
							options
						);
						console.log('refreshed session');

						// Patch the cookies in the event
						patchCookies(res.headers.getSetCookie());
					} catch (e) {
						if (e instanceof HTTPError) {
							if (await refreshTokenExpired(e.response)) {
								console.error('failed to refresh session:', await e.response.clone().text());
							}
							return e.response;
						}
						throw e;
					}

					return await ky(request, {
						...options,
						headers: {
							...options?.headers,
							Cookie: concatenateCookies(event.cookies)
						}
					});
				}
			]
		}
	};

	try {
		const fetch = ky.create(options);
		await verifySession(fetch, { role });
	} catch (e) {
		if (e instanceof HTTPError) {
			const err = await parseError(e.response);
			if (err.success && err.data.code === ApiErrorCodes.InsufficientPermissions) {
				redirect(303, '/');
			}
		}
		throw e;
	}

	return await resolve(event);
};

export const handleError: HandleServerError = async ({ error, status, message }) => {
	if (error instanceof HTTPError) {
		if (await refreshTokenExpired(error.response)) {
			redirect(303, '/login');
		}
		const err = await parseError(error.response);
		if (!err.success) {
			return {
				message,
				status: error.response.status
			};
		}
		return {
			message: err.data.message,
			errorId: err.data.error_id,
			code: err.data.code,
			status: err.data.status
		};
	}

	return {
		message,
		status
	};
};
