import { createAvatar } from '@dicebear/core';
import { shapes } from '@dicebear/collection';

/**
 * Generate a DiceBear avatar URL using the Glass collection
 * @param seed - The seed string to generate consistent avatars (e.g., team name or ID)
 * @param size - The size of the avatar in pixels (default: 40)
 * @returns SVG data URL for the generated avatar
 */
export function generateTeamAvatar(seed: string, size = 40): string {
	const avatar = createAvatar(shapes, {
		seed,
		size
	});

	return avatar.toDataUri();
}
