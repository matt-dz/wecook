import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { getUser, getUsers } from '$lib/users';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	const refreshToken = cookies.get(REFRESH_TOKEN_COOKIE_NAME);
	try {
		const [users, currentUser] = await Promise.all([
			getUsers(fetch, { access_token: accessToken }, {}, env.INTERNAL_BACKEND_URL),
			getUser(
				fetch,
				{
					headers: {
						Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}; ${REFRESH_TOKEN_COOKIE_NAME}=${refreshToken}`
					}
				},
				env.INTERNAL_BACKEND_URL
			)
		]);
		return {
			users: users,
			currentUserId: currentUser.id
		};
	} catch (e) {
		console.error('failed to get users', e);
		throw e;
	}
};
