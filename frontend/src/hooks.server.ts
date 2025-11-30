import { verifySession, refreshSession } from '$lib/auth';
import ky, { HTTPError, type Options } from 'ky';
import { baseOptions, isExpiredToken } from '$lib/http';
import { redirect, type Handle } from '@sveltejs/kit';
import * as setCookie from 'set-cookie-parser';

export const handle: Handle = async ({ event, resolve }) => {
	if (!event.route.id?.startsWith('/(app)/')) {
		const response = await resolve(event);
		return response;
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
					if (!(await isExpiredToken(response))) {
						return response;
					}

					try {
						// Refresh the session
						const res = await refreshSession({
							...baseOptions,
							headers: {
								Cookie: concatenateCookies(event.cookies)
							}
						});

						// Patch the cookies in the event
						setCookie.parse(res.headers.getSetCookie()).map(({ name, value, ...opts }) => {
							event.cookies.set(name, value, {
								...opts,
								sameSite: opts.sameSite as boolean | 'lax' | 'strict' | 'none' | undefined,
								path: opts.path ?? '/'
							});
						});
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
		redirect(303, '/');
	}

	const response = await resolve(event);
	return response;
};
