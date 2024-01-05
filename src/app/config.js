const { ipcRenderer } = require('electron');
const Store = require('electron-store');
const store = new Store();

function ConfigController($scope) {
	$scope.config = {
		publicKeyFilePath: null,
		privateKeyFilePath: null,
		keyPassphrase: null,
		passwordStoreDirectoryPath: null,
		offline: true
	};
	if(store.get('config')) {
		$scope.config = store.get('config');
	}

	$scope.addConfig = async function () {
		store.set('config', $scope.config);
		ipcRenderer.send("add:config:done", $scope.config);
	}
}


angular.module('passAppConfig', ['ngMaterial', 'ngMessages'])
	.controller('configController', ConfigController);
