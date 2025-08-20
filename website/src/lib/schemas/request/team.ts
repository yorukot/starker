import * as yup from 'yup';

export const teamSchema = yup.object({
	team_name: yup
		.string()
		.required('Team name is required')
		.min(2, 'Team name must be at least 2 characters')
		.max(50, 'Team name must be less than 50 characters')
});

export type TeamForm = yup.InferType<typeof teamSchema>;