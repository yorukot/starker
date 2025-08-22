import forge from 'node-forge';
import nacl from 'tweetnacl';

export type SSHKeyType = 'ed25519' | 'rsa';

export interface SSHKeyPair {
	privateKey: string;
	publicKey: string;
	fingerprint: string;
	keyType: SSHKeyType;
	keySize?: number;
	comment?: string;
}

export interface SSHKeyOptions {
	comment?: string;
	keySize?: number; // For RSA keys (2048, 3072, 4096)
}

export class SSHKeyError extends Error {
	constructor(
		message: string,
		public readonly keyType?: SSHKeyType
	) {
		super(message);
		this.name = 'SSHKeyError';
	}
}

function arrayToBase64(array: Uint8Array): string {
	return btoa(String.fromCharCode(...array));
}

function computeSHA256Fingerprint(publicKeyBytes: Uint8Array): string {
	const hash = forge.md.sha256.create();
	hash.update(forge.util.binary.raw.encode(publicKeyBytes));
	const hashBytes = hash.digest().getBytes();
	const hashArray = new Uint8Array(hashBytes.length);
	for (let i = 0; i < hashBytes.length; i++) {
		hashArray[i] = hashBytes.charCodeAt(i);
	}
	return 'SHA256:' + arrayToBase64(hashArray).replace(/=+$/, '');
}

function writeUint32(value: number): Uint8Array {
	const buffer = new Uint8Array(4);
	buffer[0] = (value >>> 24) & 0xff;
	buffer[1] = (value >>> 16) & 0xff;
	buffer[2] = (value >>> 8) & 0xff;
	buffer[3] = value & 0xff;
	return buffer;
}

function writeString(str: string): Uint8Array {
	const strBytes = new TextEncoder().encode(str);
	const length = writeUint32(strBytes.length);
	const result = new Uint8Array(length.length + strBytes.length);
	result.set(length, 0);
	result.set(strBytes, length.length);
	return result;
}

function writeBuffer(buffer: Uint8Array): Uint8Array {
	const length = writeUint32(buffer.length);
	const result = new Uint8Array(length.length + buffer.length);
	result.set(length, 0);
	result.set(buffer, length.length);
	return result;
}

function formatOpenSSHPrivateKey(keyData: string): string {
	const header = '-----BEGIN OPENSSH PRIVATE KEY-----';
	const footer = '-----END OPENSSH PRIVATE KEY-----';

	const lines = keyData.match(/.{1,70}/g) || [];
	const formattedKey = lines.join('\n');

	return `${header}\n${formattedKey}\n${footer}`;
}

function formatOpenSSHPublicKey(keyData: string, keyType: SSHKeyType, comment?: string): string {
	const keyTypeString = keyType === 'ed25519' ? 'ssh-ed25519' : 'ssh-rsa';
	const commentString = comment ? ` ${comment}` : '';
	return `${keyTypeString} ${keyData}${commentString}`;
}

export function generateEd25519SSHKey(options: SSHKeyOptions = {}): SSHKeyPair {
	try {
		const keyPair = nacl.sign.keyPair();
		const { publicKey, secretKey } = keyPair;

		// Extract the private key (first 32 bytes of secretKey)
		const privateKeyBytes = secretKey.slice(0, 32);

		// Create the public key in SSH wire format
		const keyTypeBytes = writeString('ssh-ed25519');
		const publicKeyBuffer = writeBuffer(publicKey);
		const publicKeyWire = new Uint8Array(keyTypeBytes.length + publicKeyBuffer.length);
		publicKeyWire.set(keyTypeBytes, 0);
		publicKeyWire.set(publicKeyBuffer, keyTypeBytes.length);

		const publicKeyBase64 = arrayToBase64(publicKeyWire);

		// Create OpenSSH private key format
		const AUTH_MAGIC = 'openssh-key-v1\0';
		const magicBytes = new TextEncoder().encode(AUTH_MAGIC);

		// Create the private key section
		const checkInt = crypto.getRandomValues(new Uint32Array(1))[0];
		const privateKeySection = new Uint8Array(1024); // Generous buffer
		let offset = 0;

		// Write check integers (twice)
		privateKeySection.set(writeUint32(checkInt), offset);
		offset += 4;
		privateKeySection.set(writeUint32(checkInt), offset);
		offset += 4;

		// Write key type
		const keyTypeString = writeString('ssh-ed25519');
		privateKeySection.set(keyTypeString, offset);
		offset += keyTypeString.length;

		// Write public key
		privateKeySection.set(writeBuffer(publicKey), offset);
		offset += writeBuffer(publicKey).length;

		// Write private key (64 bytes: 32 private + 32 public)
		const fullPrivateKey = new Uint8Array(64);
		fullPrivateKey.set(privateKeyBytes, 0);
		fullPrivateKey.set(publicKey, 32);
		privateKeySection.set(writeBuffer(fullPrivateKey), offset);
		offset += writeBuffer(fullPrivateKey).length;

		// Write comment
		const commentString = options.comment || '';
		const commentBytes = writeString(commentString);
		privateKeySection.set(commentBytes, offset);
		offset += commentBytes.length;

		// Trim to actual size
		const trimmedPrivateSection = privateKeySection.slice(0, offset);

		// Pad to block size (8 bytes)
		const blockSize = 8;
		const padding = blockSize - (trimmedPrivateSection.length % blockSize);
		const paddedPrivateSection = new Uint8Array(trimmedPrivateSection.length + padding);
		paddedPrivateSection.set(trimmedPrivateSection, 0);
		for (let i = 0; i < padding; i++) {
			paddedPrivateSection[trimmedPrivateSection.length + i] = i + 1;
		}

		// Create the full private key structure
		const cipherName = writeString('none');
		const kdfName = writeString('none');
		const kdfOptions = writeString('');
		const numberOfKeys = writeUint32(1);
		const publicKeyLength = writeUint32(publicKeyWire.length);
		const privateKeyLength = writeUint32(paddedPrivateSection.length);

		const totalLength =
			magicBytes.length +
			cipherName.length +
			kdfName.length +
			kdfOptions.length +
			numberOfKeys.length +
			publicKeyLength.length +
			publicKeyWire.length +
			privateKeyLength.length +
			paddedPrivateSection.length;

		const fullPrivateKeyBuffer = new Uint8Array(totalLength);
		offset = 0;

		fullPrivateKeyBuffer.set(magicBytes, offset);
		offset += magicBytes.length;
		fullPrivateKeyBuffer.set(cipherName, offset);
		offset += cipherName.length;
		fullPrivateKeyBuffer.set(kdfName, offset);
		offset += kdfName.length;
		fullPrivateKeyBuffer.set(kdfOptions, offset);
		offset += kdfOptions.length;
		fullPrivateKeyBuffer.set(numberOfKeys, offset);
		offset += numberOfKeys.length;
		fullPrivateKeyBuffer.set(publicKeyLength, offset);
		offset += publicKeyLength.length;
		fullPrivateKeyBuffer.set(publicKeyWire, offset);
		offset += publicKeyWire.length;
		fullPrivateKeyBuffer.set(privateKeyLength, offset);
		offset += privateKeyLength.length;
		fullPrivateKeyBuffer.set(paddedPrivateSection, offset);

		const privateKeyBase64 = arrayToBase64(fullPrivateKeyBuffer);
		const fingerprint = computeSHA256Fingerprint(publicKeyWire);

		return {
			privateKey: formatOpenSSHPrivateKey(privateKeyBase64),
			publicKey: formatOpenSSHPublicKey(publicKeyBase64, 'ed25519', options.comment),
			fingerprint,
			keyType: 'ed25519',
			comment: options.comment
		};
	} catch (error) {
		throw new SSHKeyError(
			`Failed to generate Ed25519 key: ${error instanceof Error ? error.message : 'Unknown error'}`,
			'ed25519'
		);
	}
}

export function generateRSASSHKey(options: SSHKeyOptions = {}): SSHKeyPair {
	try {
		const keySize = options.keySize || 2048;

		// Validate key size
		if (![2048, 3072, 4096].includes(keySize)) {
			throw new SSHKeyError('Invalid RSA key size. Must be 2048, 3072, or 4096 bits', 'rsa');
		}

		// Generate RSA key pair using node-forge
		const keyPair = forge.pki.rsa.generateKeyPair({ bits: keySize });
		const { privateKey, publicKey } = keyPair;

		// Use node-forge's built-in OpenSSH conversion methods
		const comment = options.comment || '';
		const privateKeySSH = forge.ssh.privateKeyToOpenSSH(privateKey);
		const publicKeySSH = forge.ssh.publicKeyToOpenSSH(publicKey, comment);

		// Create SSH wire format for fingerprint calculation
		const keyTypeBytes = writeString('ssh-rsa');

		// Extract RSA parameters for public key wire format (for fingerprint)
		const eBigInt = publicKey.e;
		const nBigInt = publicKey.n;

		// Convert BigInts to bytes for SSH wire format
		const eHex = eBigInt.toString(16);
		const nHex = nBigInt.toString(16);

		const eBytes = forge.util.hexToBytes(eHex.length % 2 ? '0' + eHex : eHex);
		const nBytes = forge.util.hexToBytes(nHex.length % 2 ? '0' + nHex : nHex);

		const eArray = new Uint8Array(eBytes.length);
		const nArray = new Uint8Array(nBytes.length);

		for (let i = 0; i < eBytes.length; i++) {
			eArray[i] = eBytes.charCodeAt(i);
		}
		for (let i = 0; i < nBytes.length; i++) {
			nArray[i] = nBytes.charCodeAt(i);
		}

		const eBuffer = writeBuffer(eArray);
		const nBuffer = writeBuffer(nArray);

		const publicKeyWire = new Uint8Array(keyTypeBytes.length + eBuffer.length + nBuffer.length);
		let offset = 0;
		publicKeyWire.set(keyTypeBytes, offset);
		offset += keyTypeBytes.length;
		publicKeyWire.set(eBuffer, offset);
		offset += eBuffer.length;
		publicKeyWire.set(nBuffer, offset);

		const fingerprint = computeSHA256Fingerprint(publicKeyWire);

		return {
			privateKey: privateKeySSH,
			publicKey: publicKeySSH,
			fingerprint,
			keyType: 'rsa',
			keySize,
			comment: options.comment
		};
	} catch (error) {
		throw new SSHKeyError(
			`Failed to generate RSA key: ${error instanceof Error ? error.message : 'Unknown error'}`,
			'rsa'
		);
	}
}

export function validateSSHKeyOptions(options: SSHKeyOptions, keyType: SSHKeyType): void {
	if (keyType === 'rsa' && options.keySize) {
		if (![2048, 3072, 4096].includes(options.keySize)) {
			throw new SSHKeyError('Invalid RSA key size. Must be 2048, 3072, or 4096 bits', 'rsa');
		}
	}

	if (options.comment && options.comment.length > 255) {
		throw new SSHKeyError('Comment must be 255 characters or less', keyType);
	}

	if (options.comment && /[\r\n]/.test(options.comment)) {
		throw new SSHKeyError('Comment cannot contain newlines', keyType);
	}
}

export function generateSSHKey(keyType: SSHKeyType, options: SSHKeyOptions = {}): SSHKeyPair {
	validateSSHKeyOptions(options, keyType);

	switch (keyType) {
		case 'ed25519':
			return generateEd25519SSHKey(options);
		case 'rsa':
			return generateRSASSHKey(options);
		default:
			throw new SSHKeyError(`Unsupported key type: ${keyType}`);
	}
}
