export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Project } from '$lib/schemas/project';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	depends(`projects:${teamID}`);

	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects`);

		if (!response.ok) {
			return {
				projects: [],
				error: `Failed to fetch projects: ${response.status}`
			};
		}

		const projects: Project[] = await response.json();

		return {
			projects
		};
	} catch (error: unknown) {
		console.error('Error fetching projects:', error);

		return {
			projects: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
