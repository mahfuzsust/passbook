const { ipcRenderer } = require('electron');
const fse = require('fs-extra');
const Store = require('electron-store');
const store = new Store();
const publicKeyFilePath = store.get('config')['publicKeyFilePath'];
const directoryPath = store.get('config')['passwordStoreDirectoryPath'];

var generator = require('generate-password');



function CredController($scope) {
    $scope.cred = {};
    $scope.editMode = false;
    ipcRenderer.on('edit:credential:value', function (event, value) {
        $scope.cred = {
            name: value.name,
            username: value.username,
            password: value.password,
            url: value.url,
            notes: value.notes,
            otpToken: value.otpToken
        }
        $scope.editMode = true;
        $scope.$apply();
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
