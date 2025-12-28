import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { getUsers } from '$lib/users';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	try {
		const users = await getUsers(
			fetch,
			{ access_token: accessToken },
			{},
			env.INTERNAL_BACKEND_URL
		);
		return {
			users: users
		};
	} catch (e) {
		console.error('failed to get users', e);
		throw e;
	}
};
