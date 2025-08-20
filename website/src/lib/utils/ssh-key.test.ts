#!/usr/bin/env node
/**
 * Simple test suite for SSH key generation functions
 * Run with: npx tsx src/lib/utils/ssh-key.test.ts
 */

import { generateSSHKey, generateEd25519SSHKey, generateRSASSHKey, SSHKeyError } from './ssh-key';

// Simple test framework
class SimpleTest {
	private passed = 0;
	private failed = 0;

	test(name: string, fn: () => void | Promise<void>) {
		try {
			const result = fn();
			if (result instanceof Promise) {
				return result.then(() => {
					console.log(`✓ ${name}`);
					this.passed++;
				}).catch((error) => {
					console.log(`✗ ${name}: ${error.message}`);
					this.failed++;
				});
			} else {
				console.log(`✓ ${name}`);
				this.passed++;
			}
		} catch (error) {
			console.log(`✗ ${name}: ${error instanceof Error ? error.message : error}`);
			this.failed++;
		}
	}

	summary() {
		const total = this.passed + this.failed;
		console.log(`\n${this.passed}/${total} tests passed`);
		if (this.failed > 0) {
			console.log(`${this.failed} tests failed`);
			process.exit(1);
		}
	}
}

const test = new SimpleTest();

// Helper function to validate SSH key format
function validateSSHKeyFormat(privateKey: string, publicKey: string, keyType: 'ed25519' | 'rsa') {
	// Validate private key format - accept both OpenSSH and traditional formats
	const validPrivateKeyHeaders = [
		'-----BEGIN OPENSSH PRIVATE KEY-----',
		'-----BEGIN RSA PRIVATE KEY-----',
		'-----BEGIN PRIVATE KEY-----'
	];
	
	const validPrivateKeyFooters = [
		'-----END OPENSSH PRIVATE KEY-----',
		'-----END RSA PRIVATE KEY-----',
		'-----END PRIVATE KEY-----'
	];
	
	const hasValidHeader = validPrivateKeyHeaders.some(header => privateKey.startsWith(header));
	// Handle both Unix (\n) and Windows (\r\n) line endings
	const hasValidFooter = validPrivateKeyFooters.some(footer => 
		privateKey.endsWith(footer) || 
		privateKey.endsWith(footer + '\n') || 
		privateKey.endsWith(footer + '\r\n')
	);
	
	if (!hasValidHeader) {
		throw new Error('Private key does not start with a valid SSH private key header');
	}
	if (!hasValidFooter) {
		throw new Error('Private key does not end with a valid SSH private key footer');
	}

	// Validate public key format
	const expectedPrefix = keyType === 'ed25519' ? 'ssh-ed25519' : 'ssh-rsa';
	if (!publicKey.startsWith(expectedPrefix)) {
		throw new Error(`Public key does not start with ${expectedPrefix}`);
	}

	// Check for base64 content
	const parts = publicKey.split(' ');
	if (parts.length < 2) {
		throw new Error('Public key format is invalid');
	}

	// Validate base64 (simple check)
	const base64Pattern = /^[A-Za-z0-9+/]+={0,2}$/;
	if (!base64Pattern.test(parts[1])) {
		throw new Error('Public key does not contain valid base64');
	}
}

// Test Ed25519 key generation
test.test('generateEd25519SSHKey - basic generation', () => {
	const keyPair = generateEd25519SSHKey();
	
	if (!keyPair.privateKey || !keyPair.publicKey) {
		throw new Error('Key pair is missing private or public key');
	}
	
	if (keyPair.keyType !== 'ed25519') {
		throw new Error(`Expected keyType 'ed25519', got '${keyPair.keyType}'`);
	}
	
	if (!keyPair.fingerprint) {
		throw new Error('Fingerprint is missing');
	}
	
	if (!keyPair.fingerprint.startsWith('SHA256:')) {
		throw new Error('Fingerprint should start with SHA256:');
	}
	  
	console.log(keyPair.privateKey)
	console.log(keyPair.publicKey)

	validateSSHKeyFormat(keyPair.privateKey, keyPair.publicKey, 'ed25519');
});

// Test Ed25519 key generation with comment
test.test('generateEd25519SSHKey - with comment', () => {
	const comment = 'test-ed25519-key';
	const keyPair = generateEd25519SSHKey({ comment });
	
	if (!keyPair.publicKey.includes(comment)) {
		throw new Error('Public key should contain the comment');
	}
	
	if (keyPair.comment !== comment) {
		throw new Error(`Expected comment '${comment}', got '${keyPair.comment}'`);
	}
});

// Test RSA key generation
test.test('generateRSASSHKey - basic generation (2048 bits)', () => {
	const keyPair = generateRSASSHKey();
	
	if (!keyPair.privateKey || !keyPair.publicKey) {
		throw new Error('Key pair is missing private or public key');
	}
	
	if (keyPair.keyType !== 'rsa') {
		throw new Error(`Expected keyType 'rsa', got '${keyPair.keyType}'`);
	}
	
	if (keyPair.keySize !== 2048) {
		throw new Error(`Expected keySize 2048, got ${keyPair.keySize}`);
	}
	
	if (!keyPair.fingerprint || !keyPair.fingerprint.startsWith('SHA256:')) {
		throw new Error('Invalid fingerprint');
	}

	console.log(keyPair.privateKey)
	console.log(keyPair.publicKey)

	validateSSHKeyFormat(keyPair.privateKey, keyPair.publicKey, 'rsa');
});

// Test RSA key generation with custom size
test.test('generateRSASSHKey - 4096 bits', () => {
	const keyPair = generateRSASSHKey({ keySize: 4096 });
	
	if (keyPair.keySize !== 4096) {
		throw new Error(`Expected keySize 4096, got ${keyPair.keySize}`);
	}
	
	console.log(keyPair.privateKey)
	console.log(keyPair.publicKey)

	validateSSHKeyFormat(keyPair.privateKey, keyPair.publicKey, 'rsa');
});

// Test RSA key generation with comment
test.test('generateRSASSHKey - with comment', () => {
	const comment = 'test-rsa-key';
	const keyPair = generateRSASSHKey({ comment, keySize: 2048 });
	
	if (!keyPair.publicKey.includes(comment)) {
		throw new Error('Public key should contain the comment');
	}

	if (keyPair.comment !== comment) {
		throw new Error(`Expected comment '${comment}', got '${keyPair.comment}'`);
	}
});

// Test generic generateSSHKey function
test.test('generateSSHKey - Ed25519', () => {
	const keyPair = generateSSHKey('ed25519', { comment: 'generic-test' });
	
	if (keyPair.keyType !== 'ed25519') {
		throw new Error(`Expected keyType 'ed25519', got '${keyPair.keyType}'`);
	}
	
	validateSSHKeyFormat(keyPair.privateKey, keyPair.publicKey, 'ed25519');
});

test.test('generateSSHKey - RSA', () => {
	const keyPair = generateSSHKey('rsa', { keySize: 2048, comment: 'generic-rsa-test' });
	
	if (keyPair.keyType !== 'rsa') {
		throw new Error(`Expected keyType 'rsa', got '${keyPair.keyType}'`);
	}
	
	if (keyPair.keySize !== 2048) {
		throw new Error(`Expected keySize 2048, got ${keyPair.keySize}`);
	}
	
	validateSSHKeyFormat(keyPair.privateKey, keyPair.publicKey, 'rsa');
});

// Test error handling
test.test('generateRSASSHKey - invalid key size', () => {
	try {
		generateRSASSHKey({ keySize: 1024 }); // Invalid size
		throw new Error('Should have thrown an error for invalid key size');
	} catch (error) {
		if (!(error instanceof SSHKeyError)) {
			throw new Error('Should throw SSHKeyError for invalid key size');
		}
		if (!error.message.includes('Invalid RSA key size')) {
			throw new Error('Error message should mention invalid key size');
		}
	}
});

test.test('generateSSHKey - unsupported key type', () => {
	try {
		// @ts-expect-error - Testing invalid key type
		generateSSHKey('invalid', {});
		throw new Error('Should have thrown an error for unsupported key type');
	} catch (error) {
		if (!(error instanceof SSHKeyError)) {
			throw new Error('Should throw SSHKeyError for unsupported key type');
		}
		if (!error.message.includes('Unsupported key type')) {
			throw new Error('Error message should mention unsupported key type');
		}
	}
});

// Test key uniqueness
test.test('generateEd25519SSHKey - keys are unique', () => {
	const keyPair1 = generateEd25519SSHKey();
	const keyPair2 = generateEd25519SSHKey();
	
	if (keyPair1.privateKey === keyPair2.privateKey) {
		throw new Error('Generated keys should be unique');
	}
	
	if (keyPair1.publicKey === keyPair2.publicKey) {
		throw new Error('Generated public keys should be unique');
	}
	
	if (keyPair1.fingerprint === keyPair2.fingerprint) {
		throw new Error('Generated fingerprints should be unique');
	}
});

test.test('generateRSASSHKey - keys are unique', () => {
	const keyPair1 = generateRSASSHKey({ keySize: 2048 });
	const keyPair2 = generateRSASSHKey({ keySize: 2048 });
	
	if (keyPair1.privateKey === keyPair2.privateKey) {
		throw new Error('Generated keys should be unique');
	}
	
	if (keyPair1.publicKey === keyPair2.publicKey) {
		throw new Error('Generated public keys should be unique');
	}
	
	if (keyPair1.fingerprint === keyPair2.fingerprint) {
		throw new Error('Generated fingerprints should be unique');
	}
});

// Run all tests
console.log('Running SSH Key Tests...\n');
test.summary();