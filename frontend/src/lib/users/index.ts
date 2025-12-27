import { PUBLIC_BACKEND_URL } from '$env/static/public';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import type { FetchType } from '$lib/http';
import type { Options } from 'ky';
import * as z from 'zod';

export const UserSchema = z.object({
	id: z.int().min(0),
	email: z.string(),
	first_name: z.string(),
	last_name: z.string(),
	role: z.enum(['user', 'admin'])
});

export type User = z.infer<typeof UserSchema>;

export type GetUserRequest = {
	access_token?: string;
};

export type GetUserResponse = User;

export async function getUser(
	fetch: FetchType,
	request: GetUserRequest,
	options?: Options
): Promise<GetUserResponse> {
	const json = await fetch
		.get(`${PUBLIC_BACKEND_URL}/api/user`, {
			...options,
			headers: {
				...options?.headers,
				Cookie: request.access_token ? `${ACCESS_TOKEN_COOKIE_NAME}=${request.access_token};` : ''
			},
			credentials: 'include'
		})
		.json();
	return UserSchema.parse(json);
}

export const GetUsersResponseSchema = z.object({
	users: z.array(UserSchema),
	cursor: z.int().min(0)
});

export type GetUsersRequest = {
	access_token?: string;
};

export type GetUsersResponse = z.infer<typeof GetUsersResponseSchema>;

export async function getUsers(
	fetch: FetchType,
	request: GetUsersRequest,
	options?: Options
): Promise<GetUsersResponse> {
	const json = await fetch
		.get(`${PUBLIC_BACKEND_URL}/api/users`, {
			...options,
			headers: {
				...options?.headers,
				Cookie: request.access_token ? `${ACCESS_TOKEN_COOKIE_NAME}=${request.access_token};` : ''
			},
			credentials: 'include'
		})
		.json();
	return GetUsersResponseSchema.parse(json);
}

export type InviteUserRequest = {
	email: string;
};

export async function inviteUser(
	fetch: FetchType,
	request: InviteUserRequest,
	options?: Options
): Promise<void> {
	await fetch.post(`${PUBLIC_BACKEND_URL}/api/user/invite`, {
		...options,
		json: request
	});
}

export type UserSignupRequest = {
	email: string;
	first_name: string;
	last_name: string;
	password: string;
	invite_code: string;
};

export async function signupRequest(
	fetch: FetchType,
	request: UserSignupRequest,
	options?: Options
): Promise<void> {
	await fetch.post(`${PUBLIC_BACKEND_URL}/api/signup`, {
		...options,
		json: request
	});
}
