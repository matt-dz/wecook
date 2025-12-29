import { type FetchType } from '$lib/http';
import ky, { type Options } from 'ky';

const ACCESS_TOKEN_COOKIE_NAME = 'access';
const REFRESH_TOKEN_COOKIE_NAME = 'refresh';
const CSRF_TOKEN_COOKIE_NAME = 'csrf';
const CSRF_HEADER = 'X-CSRF-Token';

export type Role = 'user' | 'admin';

export type VerifySessionRequest = {
	role?: Role;
};

export async function verifySession(
	fetch: FetchType,
	request: VerifySessionRequest,
	options?: Options,
	apiUrl?: string
) {
	await fetch.get(`${apiUrl ?? ''}/api/auth/verify?role=${request.role ?? 'user'}`, options);
}

export type RefreshSessionRequest = {
	refresh_token?: string;
};

// refreshSession refreshes the user session.
export async function refreshSession(
	request: RefreshSessionRequest,
	options?: Options,
	apiUrl?: string
) {
	return await ky.post(`${apiUrl ?? ''}/api/auth/refresh`, {
		...options,
		json: request,
		credentials: 'include'
	});
}

export type LoginRequest = {
	email: string;
	password: string;
};

export async function login(
	fetch: FetchType,
	request: LoginRequest,
	options?: Options,
	apiUrl?: string
) {
	return await fetch.post(`${apiUrl ?? ''}/api/login`, {
		...options,
		json: request
	});
}

export async function logout(fetch: FetchType, options?: Options, apiUrl?: string) {
	await fetch.post(`${apiUrl ?? ''}/api/logout`, {
		...options,
		credentials: 'include'
	});
}

export { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME, CSRF_TOKEN_COOKIE_NAME, CSRF_HEADER };
