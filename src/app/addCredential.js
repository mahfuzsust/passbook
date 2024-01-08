const { ipcRenderer } = require('electron');
const fse = require('fs-extra');
const Store = require('electron-store');
const store = new Store();
const publicKeyFilePath = store.get('config')['publicKeyFilePath'];
const directoryPath = store.get('config')['passwordStoreDirectoryPath'];
var generator = require('generate-password');

const getFileName = function (path) {
    path = path.replace(new RegExp('.gpg$'), '');

    let splitPath = directoryPath;
    if (directoryPath.endsWith('/') == false) {
        splitPath += '/';
    }

    const sp = path.split(splitPath);
    if (sp.length > 1) {
        return sp[1];
    }
}


function CredController($scope) {
    $scope.cred = {};
    $scope.editMode = false;
    ipcRenderer.on('edit:credential:value', function (event, value) {
        $scope.cred = {
            name: getFileName(value.path),
            username: value.username,
            password: value.password,
            url: value.url,
            notes: value.notes,
            otpToken: value.otpToken
        }
        $scope.editMode = true;
        $scope.$apply();
    });

    $scope.$watch('cred.name', function (newValue, oldValue) {
        if(!newValue || newValue === oldValue) {
            return;
        }
        const regex = /^[0-9a-zA-Z_\-./]+(?<![\/.])$/g;
        const found = newValue.match(regex);
        if (!found) {
            $scope.credForm.name.$setValidity('pattern', false);
        } else {
            $scope.credForm.name.$setValidity('pattern', true);
        }
    });

    $scope.pass = {
        length: 8,
        numbers: true,
        uppercase: true,
        symbols: true
    }

    $scope.$watch('pass', function (newValue, oldValue) {
        if (newValue !== oldValue) {
            $scope.generatePassword();
        }
    }, true);

    $scope.generatePassword = function () {
        $scope.showPassOptions = true;
        $scope.cred.password = generator.generate($scope.pass);
    }

    $scope.createEntry = async function () {
        const encrypted = await encryptGPGFile(publicKeyFilePath, createContentString($scope.cred));
        fse.outputFileSync(directoryPath + '/' + $scope.cred.name + ".gpg", encrypted);
        ipcRenderer.send("add:credential:done");
    }
}


angular.module('passAppCred', ['ngMaterial', 'ngMessages'])
    .controller('credController', CredController);
