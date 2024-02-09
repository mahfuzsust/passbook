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

function fromHTML(html, trim = true) {
    html = trim ? html.trim() : html;
    if (!html) return null;

    const template = document.createElement('template');
    template.innerHTML = html;
    const result = template.content.children;

    if (result.length === 1) return result[0];
    return result;
}

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

function MainController($scope, $interval, $mdToast, $compile) {
    const pass = "********";
    $scope.menu = [];
    $scope.stack = [];

    ipcRenderer.on('passbook:sync', function (event, value) {
        $scope.sync();
    });

    const renderListView = (arr) => {
        const fragment = document.createDocumentFragment();

        for (let i = 0; i < arr.length; i++) {
            const item = arr[i];

            let btnhtml = `
            <md-list-item class="md-2-line" ng-click="onClick(${i})">
                <md-icon class="md-avatar">`;

            if (item.directory) {
                btnhtml += `<i class="fa fa-folder"></i>`
            } else if (item.file) {
                btnhtml += `<i class="fa fa-key"></i>`
            }
            btnhtml += `</md-icon>
            <div class="md-list-item-text">
                <h3>${item.name}</h3>
            </div>
        </md-list-item>`;

            let temp = $compile(btnhtml)($scope);
            angular.element(fragment).append(temp);
        }

        $scope.menu = arr;

        let ee = document.getElementById("all__passwords");
        ee.innerHTML = '';
        angular.element(ee).append(fragment);
    };

    // document.getElementById("all__passwords").addEventListener("scrollend", (event) => {
    //     console.log(event);
    // });



    let arr = [];
    getAllCredentials(directoryPath, arr);

    renderListView(arr);

    let intervalPromise;

    const setRemainingColor = (remainingTime) => {
        if ($scope.cred.remainingTime >= 20) {
            $scope.remainingColor = 'blue';
        } else if ($scope.cred.remainingTime >= 10) {
            $scope.remainingColor = 'green';
        } else $scope.remainingColor = 'orange';
    }

    $scope.onClick = async function (kk) {
        const idx = Number(kk);
        const item = $scope.menu[idx];
        $scope.password = pass;
        $scope.cred = null;
        if (intervalPromise) {
            $interval.cancel(intervalPromise);
        }

        if (item.directory) {
            $scope.stack.push($scope.menu);
            renderListView(item.child);
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
        renderListView($scope.stack.pop());
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

        let arr = [];
        getAllCredentials(directoryPath, arr);
        renderListView(arr);
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
            renderListView($scope.stack.pop());
            return;
        }
        if ($scope.searchTerm.length < 3) {
            return;
        }
        const searchTerm = $scope.searchTerm;
        const searchResults = fuzzysort.go(searchTerm, flatNames, { key: 'name' });
        if ($scope.stack.length == 0)
            $scope.stack.push($scope.menu);
        renderListView(searchResults.map(x => x.obj));
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
        let arr = [];
        getAllCredentials(directoryPath, arr);
        renderListView(arr);
    }
    $scope.showRaw = function (val) {
        $scope.rawVisible = val;
    }

}


angular.module('passApp', ['ngMaterial'])
    .controller('mainController', MainController);