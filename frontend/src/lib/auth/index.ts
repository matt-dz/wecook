import { type FetchType } from '$lib/http';
import ky, { type Options } from 'ky';
import { PUBLIC_BACKEND_URL } from '$env/static/public';

const ACCESS_TOKEN_COOKIE_NAME = 'access';
const REFRESH_TOKEN_COOKIE_NAME = 'refresh';

export async function verifySession(fetch: FetchType, options?: Options) {
	await fetch.get(`${PUBLIC_BACKEND_URL}/api/auth/verify`, options);
}

export type RefreshSessionRequest = {
	refresh_token?: string;
};

// refreshSession refreshes the user session.
export async function refreshSession(request: RefreshSessionRequest, options?: Options) {
	return await ky.post(`${PUBLIC_BACKEND_URL}/api/auth/refresh`, {
		...options,
		json: request,
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
