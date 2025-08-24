export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { LayoutLoad } from './$types';
import type { Team } from '$lib/schemas/team';
import type { Project } from '$lib/schemas/project';
import type { Service } from '$lib/schemas/service';

export const load: LayoutLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	const serviceID = params.serviceID;
	depends(`team:${teamID}`);
	depends(`project:${projectID}`);
	depends(`service:${serviceID}`);

	try {
		// Fetch team, project, and service data in parallel
		const [teamResponse, projectResponse, serviceResponse] = await Promise.all([
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}`),
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}`),
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}`)
		]);

		if (!teamResponse.ok) {
			return {
				team: null,
				project: null,
				service: null,
				error: `Failed to fetch team: ${teamResponse.status}`
			};
		}

		if (!projectResponse.ok) {
			return {
				team: null,
				project: null,
				service: null,
				error: `Failed to fetch project: ${projectResponse.status}`
			};
		}

		if (!serviceResponse.ok) {
			return {
				team: null,
				project: null,
				service: null,
				error: `Failed to fetch service: ${serviceResponse.status}`
			};
		}

		const team: Team = await teamResponse.json();
		const project: Project = await projectResponse.json();
		const service: Service = await serviceResponse.json();

		return {
			team,
			project,
			service
		};
	} catch (error: unknown) {
		console.error('Error fetching team, project, and service data:', error);

		return {
			team: null,
			project: null,
			service: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
