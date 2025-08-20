export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { LayoutLoad } from './$types';
import type { Team } from '$lib/schemas/team';

export const load: LayoutLoad = async ({ params, depends }) => {
  const teamID = params.teamID;
  depends('team:current');
  
  try {
    const response = await authGet(`${PUBLIC_API_BASE_URL}/teams`);
    
    if (!response.ok) {
      return {
        teams: [],
        currentTeam: null,
        error: `Failed to fetch teams: ${response.status}`
      };
    }

    const teams: Team[] = await response.json();
    const currentTeam = teams.find(team => team.id === teamID) || null;

    return {
      teams,
      currentTeam,
    };
  } catch (error: unknown) {
    console.error('Error fetching teams:', error);
    
    return {
      teams: [],
      currentTeam: null,
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
};
