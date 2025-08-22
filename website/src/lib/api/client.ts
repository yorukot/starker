import { browser } from '$app/environment';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { getValidToken } from './auth.js';

/**
 * Custom fetch wrapper that automatically includes Bearer token authentication
 * and handles token refresh when needed
 */
export async function authFetch(url: string | URL, options: RequestInit = {}): Promise<Response> {
	if (!browser) {
		throw new Error('authFetch can only be used in the browser');
	}

	// Get a valid token (will refresh if needed)
	const token = await getValidToken();

	// Merge headers with Authorization header
	const headers = new Headers(options.headers);
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}

	// Set default content type if not specified
	if (!headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	// Make the request
	const response = await fetch(url, {
		...options,
		headers
	});

	return response;
}

/**
 * POST request with Bearer token authentication
 */
export async function authPost(url: string, data?: unknown): Promise<Response> {
	return authFetch(url, {
		method: 'POST',
		body: data ? JSON.stringify(data) : undefined
	});
}

/**
 * GET request with Bearer token authentication
 */
export async function authGet(url: string): Promise<Response> {
	return authFetch(url, {
		method: 'GET'
	});
}

/**
 * PUT request with Bearer token authentication
 */
export async function authPut(url: string, data?: unknown): Promise<Response> {
	return authFetch(url, {
		method: 'PUT',
		body: data ? JSON.stringify(data) : undefined
	});
}

/**
 * PATCH request with Bearer token authentication
 */
export async function authPatch(url: string, data?: unknown): Promise<Response> {
	return authFetch(url, {
		method: 'PATCH',
		body: data ? JSON.stringify(data) : undefined
	});
}

/**
 * DELETE request with Bearer token authentication
 */
export async function authDelete(url: string): Promise<Response> {
	return authFetch(url, {
		method: 'DELETE'
	});
}

/**
 * Creates a Felte-compatible fetch function with Bearer token authentication
 * This is the key function for integrating with Felte forms
 */
export function createAuthenticatedFetch() {
	return async (url: string | URL, options: RequestInit = {}) => {
		// Ensure we have a full URL
		const fullUrl =
			typeof url === 'string' && !url.startsWith('http') ? `${PUBLIC_API_BASE_URL}${url}` : url;

		return authFetch(fullUrl, options);
	};
}
