import type { LayoutServerLoad } from './$types';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';

export const load: LayoutServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);

	return {
		isLoggedIn: !!accessToken
	};
};
