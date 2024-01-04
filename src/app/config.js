const { ipcRenderer } = require('electron');

function ConfigController($scope) {
	$scope.config = {
		publicKeyFilePath: null,
		privateKeyFilePath: null,
		keyPassphrase: null,
		passwordStoreDirectoryPath: null,
		gitSshUrl: null
	};

	$scope.addConfig = async function () {
		ipcRenderer.send("add:config:done", $scope.config);
	}
}


angular.module('passAppConfig', ['ngMaterial', 'ngMessages'])
	.controller('configController', ConfigController);
