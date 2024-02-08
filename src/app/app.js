const { ipcRenderer } = require('electron');
const fs = require('fs');
const path = require('path');
const fuzzysort = require('fuzzysort');
const copyPaste = require('copy-paste');
const Store = require('electron-store');
const store = new Store();
const privateKeyFilePath = store.get('config')['privateKeyFilePath'];
const publicKeyFilePath = store.get('config')['publicKeyFilePath'];
const directoryPath = store.get('config')['passwordStoreDirectoryPath'];
const passphrase = store.get('config')['keyPassphrase'];
const offline = store.get('config')['offline'];

const simpleGit = require('simple-git');
const git = simpleGit(directoryPath, { binary: 'git' });

const flatNames = [];

const getAllCredentials = function (directoryPath, arr) {
    const items = fs.readdirSync(directoryPath);
    items.forEach(item => {
        if (item.startsWith('.')) {
            return;
        }
        const itemPath = path.join(directoryPath, item);
        const stats = fs.statSync(itemPath);
        if (stats.isFile()) {
            let x = {
                name: item.split('.gpg')[0],
                path: itemPath,
                file: true
            };
            arr.push(x);
            flatNames.push(x);
        } else if (stats.isDirectory()) {
            let x = {
                name: item,
                path: itemPath,
                directory: true,
                child: []
            }
            arr.push(x);
            getAllCredentials(itemPath, x.child);
        }
    });
}

function MainController($scope, $interval, $mdToast) {
    const pass = "********";
    $scope.menu = [];
    $scope.stack = [];

    ipcRenderer.on('passbook:sync', function (event, value) {
        $scope.sync();
    });

    getAllCredentials(directoryPath, $scope.menu);

    let intervalPromise;

    const setRemainingColor = (remainingTime) => {
        if ($scope.cred.remainingTime >= 20) {
            $scope.remainingColor = 'blue';
        } else if ($scope.cred.remainingTime >= 10) {
            $scope.remainingColor = 'green';
        } else $scope.remainingColor = 'orange';
    }

    $scope.onClick = async function (item) {
        $scope.password = pass;
        $scope.cred = null;
        if (intervalPromise) {
            $interval.cancel(intervalPromise);
        }

        if (item.directory) {
            $scope.stack.push($scope.menu);
            $scope.menu = item.child;
        } else if (item.file) {
            const content = await decryptGPGFile(item.path, privateKeyFilePath, publicKeyFilePath, passphrase);
            $scope.cred = getCredentialObject(content, item.path, item.name);
            $scope.cred.name = item.name;
            $scope.cred.path = item.path;

            if ($scope.cred.otp && $scope.cred.remainingTime != null) {
                setRemainingColor($scope.cred.remainingTime);
                intervalPromise = $interval(function () {
                    $scope.cred.remainingTime -= 1;
                    setRemainingColor($scope.cred.remainingTime);

                    if ($scope.cred.remainingTime <= 0) {
                        $scope.cred.otp = generateOTP($scope.cred.otpUrl);
                        $scope.cred.remainingTime = 30;
                        setRemainingColor($scope.cred.remainingTime);
                    }

                }, 1000);
            }
            $scope.$applyAsync();
        }
    }
    $scope.back = function (msg) {
        $scope.menu = $scope.stack.pop();
    }

    $scope.sync = async function () {
        showToast($mdToast, 'Syncing...');
        if (!offline) {
            const status = await git.status();

            if (status.isClean()) {
                showToast($mdToast, 'Updating from remote...');
                await git.pull();
            } else {
                showToast($mdToast, 'Git commit...');
                await git.commit('Updated at ' + new Date().toLocaleString());

                showToast($mdToast, 'Updating from remote...');
                await git.add('.');
                
                showToast($mdToast, 'Git push...');
                await git.push();
            }
        }
        $scope.menu = [];
        getAllCredentials(directoryPath, $scope.menu);
        $scope.$applyAsync();
        showToast($mdToast, 'Sync completed!');
    }

    $scope.showPassword = function (cred) {
        cred.showPassword = !cred.showPassword;
        $scope.password = cred.password;
    }
    $scope.hidePassword = function (cred) {
        cred.showPassword = !cred.showPassword;
        $scope.password = pass;
    }
    $scope.search = function () {
        if (!$scope.searchTerm || $scope.searchTerm.length === 0) {
            if ($scope.stack.length === 0) {
                return;
            }
            $scope.menu = $scope.stack.pop();
            return;
        }
        if ($scope.searchTerm.length < 3) {
            return;
        }
        const searchTerm = $scope.searchTerm;
        const searchResults = fuzzysort.go(searchTerm, flatNames, { key: 'name' })
        $scope.stack.push($scope.menu);
        $scope.menu = searchResults.map(x => x.obj);
    }
    $scope.copyToClipboard = function (text) {
        showToast($mdToast, 'Copied to clipboard!');
        copyPaste.copy(text);
    }

    $scope.addCredential = function () {
        ipcRenderer.send("add:credential", "hello");
    }

    $scope.editCredential = function (item) {
        ipcRenderer.send("edit:credential", item);
    }
    $scope.deleteCredential = function (item) {
        fs.unlinkSync(item.path);
        $scope.cred = null;
        $scope.menu = [];
        getAllCredentials(directoryPath, $scope.menu);
    }
    $scope.showRaw = function (val) {
        $scope.rawVisible = val;
    }

}


angular.module('passApp', ['ngMaterial'])
    .controller('mainController', MainController);