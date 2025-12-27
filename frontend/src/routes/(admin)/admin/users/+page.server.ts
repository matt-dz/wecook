import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { getUsers } from '$lib/users';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	try {
		const users = await getUsers(fetch, { access_token: accessToken });
		return {
			users: users
		};
	} catch (e) {
		console.error('failed to get users', e);
		throw e;
	}
};
