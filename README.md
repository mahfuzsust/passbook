# passbook

GUI application to support [pass](https://www.passwordstore.org/) with totp integration. Application written in [Electron](https://www.electronjs.org/).


## Setup 

### GPG setup

[Github tutorial](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)

1. Download and install the [GPG command line tools](https://www.gnupg.org/download) for your operating system. 
2. ```gpg --full-generate-key```
3. ```gpg --list-secret-keys --keyid-format=long ```
4. ``` gpg -o <PUBLIC_KEY_FILE_PATH.asc> --armor --export <KEY_ID>```
5. ``` gpg -o <PRIVATE_KEY_FILE_PATH.asc> --armor --export-secret-key <KEY_ID>```

### Pass setup (Recommended)
Follow the tutorial for [pass](https://www.passwordstore.org/) to install

After the installation is done
1. ``` pass init <KEY_ID>```
2. ``` pass git init ```

After everything is done the password storage path will be `/Users/<username>/.password-store`

### Without pass
Any folder where passwords will be stored

## Git integration
1. Initialize git
    * for pass installed, run `pass git init`
    * otherwise run `git init` inside the folder
2. Create a private repository in Github/Bitbucket/Gitlab
3. Set the ssh remote url. For example, run `git remote add origin git@github.com:<username>/<repo>.git`. For ssh integration with git, follow github [tutorial](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)

## Application config
1. set the gpg public key file path (path stored in-memory and not synced, changing the file path will not work)
2. set the gpg private key file path (path stored in-memory and not synced, changing the file path will not work)
3. Passphrase during gpg key generation (path stored in-memory and not synced)
4. password storage path 
    * for pass `/Users/<username>/.password-store`
    * otherwise the folder path
5. offline / online


## License

[License](LICENSE)
