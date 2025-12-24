import { verifySession, refreshSession, ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import ky, { HTTPError, type Options } from 'ky';
import { baseOptions } from '$lib/http';
import { REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import { accessTokenExpired, refreshTokenExpired } from '$lib/errors/api';
import { redirect, type Handle } from '@sveltejs/kit';
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

	if (event.route.id === '/login') {
		return await resolve(event);
	}

	// Refresh session if necessary
	const accessToken = event.cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	const refreshToken = event.cookies.get(REFRESH_TOKEN_COOKIE_NAME);
	if (!accessToken && refreshToken) {
		try {
			const res = await refreshSession({
				refresh_token: refreshToken
			});
			patchCookies(res.headers.getSetCookie());
		} catch (e) {
			console.error('failed to refresh session', e);
			if (event.route.id !== '/login') {
				redirect(303, '/login');
			}
		}
	}

	// Exit early, no auth required
	if (!event.route.id?.startsWith('/(user)/')) {
		return await resolve(event);
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
		await verifySession(fetch);
	} catch (e) {
		if (e instanceof HTTPError) {
			if (await refreshTokenExpired(e.response)) {
				redirect(303, '/login');
			}
			console.error(e.message);
			console.error(await e.response.json());
		} else {
			console.error(e);
		}
		redirect(303, '/');
	}

	const response = await resolve(event);
	return response;
};
