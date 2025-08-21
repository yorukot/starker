export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Server } from '$lib/schemas/server';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	depends(`servers:${teamID}`);

	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/servers`);

		if (!response.ok) {
			return {
				servers: [],
				error: `Failed to fetch servers: ${response.status}`
			};
		}

		const servers: Server[] = await response.json();

		return {
			servers
		};
	} catch (error: unknown) {
		console.error('Error fetching servers:', error);

		return {
			servers: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};