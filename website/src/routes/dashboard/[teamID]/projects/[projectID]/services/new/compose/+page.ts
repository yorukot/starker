export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Project } from '$lib/schemas/project';
import type { Server } from '$lib/schemas/server';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	depends(`project:${projectID}`);

	try {
		// Fetch both project and servers data in parallel
		const [projectResponse, serversResponse] = await Promise.all([
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}`),
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/servers`)
		]);

		if (!projectResponse.ok) {
			return {
				project: null,
				servers: [],
				error: `Failed to fetch project: ${projectResponse.status}`
			};
		}

		if (!serversResponse.ok) {
			return {
				project: await projectResponse.json(),
				servers: [],
				error: `Failed to fetch servers: ${serversResponse.status}`
			};
		}

		const project: Project = await projectResponse.json();
		const servers: Server[] = await serversResponse.json();

		return {
			project,
			servers
		};
	} catch (error: unknown) {
		console.error('Error fetching data:', error);

		return {
			project: null,
			servers: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
