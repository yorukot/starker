export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Service, ServiceContainer } from '$lib/schemas/service';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	const serviceID = params.serviceID;
	depends(`service:${serviceID}:logs`);

	try {
		// Fetch service containers
		const containersResponse = await authGet(
			`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/containers`
		);

		if (!containersResponse.ok) {
			return {
				service: null,
				containers: [],
				error: `Failed to fetch containers: ${containersResponse.status}`
			};
		}

		const containersData = await containersResponse.json();
		// Handle both wrapped (containersData.data) and direct array responses
		const containers: ServiceContainer[] = Array.isArray(containersData)
			? containersData
			: containersData?.data || [];

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
			containers
		};
	} catch (error: unknown) {
		console.error('Error fetching logs data:', error);

		return {
			service: null,
			containers: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
