import type { PageServerLoad } from './$types';
import { getPreferences } from '$lib/admin';
import fetch from '$lib/http';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	try {
		const preferences = await getPreferences(
			fetch,
			{
				headers: {
					Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}`
				}
			},
			env.INTERNAL_BACKEND_URL
		);
		return { preferences };
	} catch (e) {
		console.error('failed to get preferences', e);
		throw e;
	}
};
