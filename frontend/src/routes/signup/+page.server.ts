import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const load: PageServerLoad = ({ url }) => {
	const code = url.searchParams.get('code');
	if (code === null) {
		redirect(303, '/login');
	}

	return {
		code
	};
};
