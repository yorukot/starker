export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { ServiceEnvironment } from '$lib/schemas/service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const { teamID, projectID, serviceID } = params;

	try {
		const response = await authGet(
			`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/env`
		);

		if (!response.ok) {
			const errorData = await response.json();
			return {
				environments: [],
				error: errorData.message || 'Failed to load environment variables'
			};
		}

		const data = await response.json();

		// Handle both possible response formats: direct array or wrapped in data property
		let environments: ServiceEnvironment[] = [];
		if (Array.isArray(data)) {
			environments = data as ServiceEnvironment[];
		} else if (data?.data && Array.isArray(data.data)) {
			environments = data.data as ServiceEnvironment[];
		}

		return {
			environments,
			error: null
		};
	} catch (error) {
		console.error('Error loading environment variables:', error);
		return {
			environments: [],
			error: 'Failed to load environment variables. Please try again.'
		};
	}
};
