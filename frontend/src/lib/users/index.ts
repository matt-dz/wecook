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

export type GetUserResponse = User;

export async function getUser(
	fetch: FetchType,
	options?: Options,
	apiUrl?: string
): Promise<GetUserResponse> {
	const json = await fetch
		.get(`${apiUrl ?? ''}/api/user`, {
			...options,
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
	options?: Options,
	apiUrl?: string
): Promise<GetUsersResponse> {
	const json = await fetch
		.get(`${apiUrl ?? ''}/api/users`, {
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
	options?: Options,
	apiUrl?: string
): Promise<void> {
	await fetch.post(`${apiUrl ?? ''}/api/user/invite`, {
		...options,
		json: request
	});
}

export type UserSignupRequest = {
	email: string;
	first_name: string;
	last_name: string;
	password: string;
	invite_code?: string;
};

export async function signupRequest(
	fetch: FetchType,
	request: UserSignupRequest,
	options?: Options,
	apiUrl?: string
): Promise<void> {
	await fetch.post(`${apiUrl ?? ''}/api/signup`, {
		...options,
		json: request
	});
}

export type ChangePasswordRequest = {
	current_password: string;
	new_password: string;
};

export async function changePassword(
	fetch: FetchType,
	request: ChangePasswordRequest,
	options?: Options,
	apiUrl?: string
): Promise<void> {
	await fetch.patch(`${apiUrl ?? ''}/api/user/password`, {
		...options,
		json: request
	});
}
