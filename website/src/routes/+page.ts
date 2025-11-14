export const ssr = false;

import { goto } from '$app/navigation';
import { getValidToken } from '$lib/api/auth';

export const load = async () => {
    const token = await getValidToken();
    if (!token) {
        goto('/auth/login');
        return {};
    } else {
        goto('/dashboard');
        return {};
    }
};
