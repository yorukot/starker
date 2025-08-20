// Provider enum matching Go models
export enum Provider {
	EMAIL = 'email',
	GOOGLE = 'google'
}

// User type
export interface User {
	id: string;
	display_name: string;
	avatar?: string;
	created_at: string;
	updated_at: string;
}

// Account type
export interface Account {
	id: string;
	provider: Provider;
	provider_user_id: string;
	user_id: string;
	email: string;
	created_at: string;
	updated_at: string;
}

// OAuth Token type
export interface OAuthToken {
	account_id: string;
	access_token: string;
	refresh_token?: string;
	expiry: string;
	token_type: string;
	provider: Provider;
	created_at: string;
	updated_at: string;
}

// Refresh Token type
export interface RefreshToken {
	id: string;
	user_id: string;
	token: string;
	user_agent?: string;
	ip?: string;
	used_at?: string;
	created_at: string;
}