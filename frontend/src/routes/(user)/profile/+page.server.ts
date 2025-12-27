import type { PageServerLoad } from './$types';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import fetch from '$lib/http';
import { getUser } from '$lib/users';
import { redirect } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		console.log('no access token available, sending user back to login.');
		redirect(307, '/login');
	}
	const refreshToken = cookies.get(REFRESH_TOKEN_COOKIE_NAME);

	const user = await getUser(fetch, {
		headers: {
			Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}; ${REFRESH_TOKEN_COOKIE_NAME}=${refreshToken}`
		}
	});
	return {
		user
	};
};
