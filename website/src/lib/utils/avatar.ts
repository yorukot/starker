import { createAvatar } from '@dicebear/core';
import { shapes, thumbs } from '@dicebear/collection';

export function generateTeamAvatar(seed: string, size = 40): string {
	const avatar = createAvatar(shapes, {
		seed,
		size
	});

	return avatar.toDataUri();
}

export function generateUserAvatar(seed: string, size = 40): string {
	const avatar = createAvatar(thumbs, {
		seed,
		size
	});

	return avatar.toDataUri();
}
