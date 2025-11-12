export const ssr = false;

import { goto } from '$app/navigation';
import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { Team } from '$lib/schemas/team';

export const load = async () => {
    try {
        const response = await authGet(`${PUBLIC_API_BASE_URL}/teams`);
        if (!response.ok) {
            return { teams: [], error: `Failed to fetch teams: ${response.status}` };
        }

        const teams: Team[] = await response.json();

        if (teams.length === 0) {
            return { teams };
        } else {
            goto(`/dashboard/${teams[0].id}/projects`);
        }
    } catch (error) {
        console.error('Error fetching teams:', error);
        return { teams: [], error: 'Failed to fetch teams' };
    }

    return {};
};
