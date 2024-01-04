const openpgp = require('openpgp');
const fs = require('fs');

async function encryptGPGFile(publicKeyPath, content) {
    try {
        const publicKey = await openpgp.readKey({ armoredKey: fs.readFileSync(publicKeyPath, 'utf-8') });

        const encrypted = await openpgp.encrypt({
            message: await openpgp.createMessage({ text: content }),
            format: 'binary',
            encryptionKeys: publicKey,
        });

        return encrypted;
    } catch (error) {
        console.error('Error:', error.message);
    }
}

async function decryptGPGFile(filePath, privateKeyPath, publicKeyPath, passphrase) {
    try {
        const encryptedData = fs.readFileSync(filePath);

        const privateKey = await openpgp.decryptKey({
            privateKey: await openpgp.readPrivateKey({ armoredKey: fs.readFileSync(privateKeyPath, 'utf-8') }),
            passphrase: passphrase,
        });

        const publicKey = await openpgp.readKey({ armoredKey: fs.readFileSync(publicKeyPath, 'utf-8') });

        const message = await openpgp.readMessage({ binaryMessage: encryptedData });

        const { data: decrypted } = await openpgp.decrypt({
            message: message,
            verificationKeys: publicKey,
            decryptionKeys: privateKey
        });
        return decrypted;
    } catch (error) {
        console.error('Error:', error.message);
    }
}