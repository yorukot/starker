export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Service, ServiceComposeConfig } from '$lib/schemas/service';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	const serviceID = params.serviceID;
	depends(`service:${serviceID}:compose`);

	try {
		// Fetch service compose configuration
		const composeResponse = await authGet(
			`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/compose`
		);

		if (!composeResponse.ok) {
			return {
				service: null,
				composeConfig: null,
				error: `Failed to fetch compose configuration: ${composeResponse.status}`
			};
		}

		const composeConfig: ServiceComposeConfig = await composeResponse.json();

		// Also fetch basic service info for display
		const serviceResponse = await authGet(
			`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}`
		);

		let service: Service | null = null;
		if (serviceResponse.ok) {
			service = await serviceResponse.json();
		}

		return {
			service,
			composeConfig
		};
	} catch (error: unknown) {
		console.error('Error fetching compose data:', error);

		return {
			service: null,
			composeConfig: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
