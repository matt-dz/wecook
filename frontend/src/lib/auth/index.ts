import { type FetchType } from '$lib/http';
import ky, { type Options } from 'ky';
import { PUBLIC_BACKEND_URL } from '$env/static/public';

const ACCESS_TOKEN_COOKIE_NAME = (import.meta.env.PROD ? '__Host-Http-' : '') + 'access';
const REFRESH_TOKEN_COOKIE_NAME = (import.meta.env.PROD ? '__Host-Http-' : '') + 'refresh';

export async function verifySession(fetch: FetchType, options?: Options) {
	await fetch.get(`${PUBLIC_BACKEND_URL}/api/auth/session/verify`, options);
}

// refreshSession refreshes the user session.
export async function refreshSession(options?: Options) {
	return await ky.post(`${PUBLIC_BACKEND_URL}/api/auth/session/refresh`, {
		...options,
		credentials: 'include'
	});
}

export type LoginRequest = {
	email: string;
	password: string;
};

export async function login(fetch: FetchType, request: LoginRequest, options?: Options) {
	return await fetch.post(`${PUBLIC_BACKEND_URL}/api/login`, {
		...options,
		json: request
	});
}

export { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME };
