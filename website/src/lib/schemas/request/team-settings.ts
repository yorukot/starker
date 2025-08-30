import * as yup from 'yup';

export const teamSettingsSchema = yup.object({
	name: yup
		.string()
		.required('Team name is required')
		.min(2, 'Team name must be at least 2 characters')
		.max(50, 'Team name must be less than 50 characters')
		.trim()
});

export type TeamSettingsForm = yup.InferType<typeof teamSettingsSchema>;

export const deleteTeamSchema = yup.object({
	confirmText: yup
		.string()
		.required('Please type the team name to confirm')
		.test('match-team-name', 'Team name does not match', function (value) {
			const teamName = this.options.context?.teamName;
			return value === teamName;
		})
});

export type DeleteTeamForm = yup.InferType<typeof deleteTeamSchema>;
