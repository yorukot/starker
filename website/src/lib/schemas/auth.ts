import * as v from 'valibot';

export const registerSchema = v.pipe(
	v.object({
		displayName: v.pipe(
			v.string('Your display name must be a string.'),
			v.minLength(1, 'Display name is required')
		),
		email: v.pipe(
			v.string('Email must be a string.'),
			v.minLength(1, 'Email is required'),
			v.email('Please enter a valid email address.')
		),
		password: v.pipe(
			v.string('Password must be a string.'),
			v.minLength(1, 'Password is required'),
			v.minLength(8, 'Password must be at least 8 characters'),
			v.regex(/^(?=.*[A-Z])(?=.*[a-z])(?=.*[0-9]).+$/, 'Password must contain at least one uppercase letter, one lowercase letter, and one number')
		),
		confirmPassword: v.pipe(
			v.string('Confirm password must be a string.'),
			v.minLength(1, 'Please confirm your password')
		)
	}),
	v.forward(
		v.partialCheck(
			[['password'], ['confirmPassword']],
			(input) => input.password === input.confirmPassword,
			'Passwords do not match'
		),
		['confirmPassword']
	)
);

export type RegisterForm = v.InferInput<typeof registerSchema>;