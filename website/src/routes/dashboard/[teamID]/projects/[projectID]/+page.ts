export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Project } from '$lib/schemas/project';
import type { Service } from '$lib/schemas/service';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	depends(`project:${projectID}`);
	depends(`services:${projectID}`);

	try {
		// Fetch project and services in parallel
		const [projectResponse, servicesResponse] = await Promise.all([
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}`),
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services`)
		]);

		if (!projectResponse.ok) {
			return {
				project: null,
				services: [],
				error: `Failed to fetch project: ${projectResponse.status}`
			};
		}

		const project: Project = await projectResponse.json();

		// Services might not exist yet, so handle 404 gracefully
		let services: Service[] = [];
		if (servicesResponse.ok) {
			services = await servicesResponse.json();
		} else if (servicesResponse.status !== 404) {
			console.warn('Failed to fetch services:', servicesResponse.status);
		}

		return {
			project,
			services
		};
	} catch (error: unknown) {
		console.error('Error fetching project data:', error);

		return {
			project: null,
			services: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
