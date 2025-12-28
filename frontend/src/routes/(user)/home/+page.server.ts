import type { PageServerLoad } from './$types';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import fetch from '$lib/http';
import { getPersonalRecipes } from '$lib/recipes';
import { redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		console.log('no access token available, sending user back to login.');
		redirect(307, '/login');
	}
	const refreshToken = cookies.get(REFRESH_TOKEN_COOKIE_NAME);

	try {
		const recipes = await getPersonalRecipes(
			fetch,
			{
				headers: {
					Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}; ${REFRESH_TOKEN_COOKIE_NAME}=${refreshToken}`
				}
			},
			env.INTERNAL_BACKEND_URL
		);
		return {
			recipes
		};
	} catch (e) {
		console.error(e);
		throw e;
	}
};
