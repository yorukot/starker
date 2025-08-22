export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Project } from '$lib/schemas/project';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	depends(`project:${projectID}`);

	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}`);

		if (!response.ok) {
			return {
				project: null,
				error: `Failed to fetch project: ${response.status}`
			};
		}

		const project: Project = await response.json();

		return {
			project
		};
	} catch (error: unknown) {
		console.error('Error fetching project:', error);

		return {
			project: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};