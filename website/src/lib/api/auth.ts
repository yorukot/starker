import { goto } from '$app/navigation';
import { browser } from '$app/environment';

import { PUBLIC_API_BASE_URL } from '$env/static/public';

export interface RefreshTokenResponse {
	access_token: string;
}

export interface ApiError {
	err_code: string;
	message: string;
}

/**
 * Refreshes the access token using the refresh token stored in httpOnly cookies
 * Sets the new access token in sessionStorage with 15-minute expiration
 * Redirects to login page if refresh fails
 */
export async function refreshToken(): Promise<string | null> {
	if (!browser) return null;

	try {
		const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/refresh`, {
			method: 'POST',
			credentials: 'include', // Include httpOnly refresh token cookie
			headers: {
				'Content-Type': 'application/json'
			}
		});

		if (!response.ok) {
			// If refresh fails, redirect to login
			sessionStorage.removeItem('access_token');
			sessionStorage.removeItem('token_expiry');
			await goto('/auth/login');
			return null;
		}

		const data: RefreshTokenResponse = await response.json();

		// Store access token in sessionStorage with 15-minute expiration
		const expiryTime = Date.now() + 10 * 60 * 1000; // 15 minutes
		sessionStorage.setItem('access_token', data.access_token);
		sessionStorage.setItem('token_expiry', expiryTime.toString());

		return data.access_token;
	} catch (error) {
		console.error('Token refresh failed:', error);
		// On error, redirect to login
		sessionStorage.removeItem('access_token');
		sessionStorage.removeItem('token_expiry');
		await goto('/auth/login');
		return null;
	}
}

/**
 * Checks if the current access token is valid and not expired
 */
export function isTokenValid(): boolean {
	if (!browser) return false;

	const token = sessionStorage.getItem('access_token');
	const expiry = sessionStorage.getItem('token_expiry');

	if (!token || !expiry) return false;

	return Date.now() < parseInt(expiry);
}

/**
 * Gets the current access token, refreshing if necessary
 */
export async function getValidToken(): Promise<string | null> {
	if (!browser) return null;

	if (isTokenValid()) {
		return sessionStorage.getItem('access_token');
	}

	// Token is expired or missing, try to refresh
	return await refreshToken();
}
