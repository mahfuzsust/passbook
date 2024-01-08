const { ipcRenderer } = require('electron');
const Store = require('electron-store');
const store = new Store();
const fs = require('fs');

function ConfigController($scope) {
	$scope.config = {
		publicKeyFilePath: null,
		privateKeyFilePath: null,
		keyPassphrase: null,
		passwordStoreDirectoryPath: null,
		offline: true
	};
	if (store.get('config')) {
		$scope.config = store.get('config');
	}

	function isFile(path) {
		try {
			if (!fs.existsSync(path)) {
				return false;
			}
			const stats = fs.statSync(path);
			return stats.isFile();
		} catch (error) {
			return false;
		}
	}
	function isDirectory(path) {
		try {
			if (!fs.existsSync(path)) {
				return false;
			}
			const stats = fs.statSync(path);
			return stats.isDirectory();
		} catch (error) {
			return false;
		}
	}


	$scope.$watch('config', (newValue, oldValue) => {
		if (!newValue || newValue === oldValue) {
			return;
		}
		if (!isFile(newValue.publicKeyFilePath)) {
			$scope.configForm.publicKeyFilePath.$setValidity('validity', false);
		} else {
			$scope.configForm.publicKeyFilePath.$setValidity('validity', true);
		}

		if (!isFile(newValue.privateKeyFilePath)) {
			$scope.configForm.privateKeyFilePath.$setValidity('validity', false);
		} else {
			$scope.configForm.privateKeyFilePath.$setValidity('validity', true);
		}

		if (!isDirectory(newValue.passwordStoreDirectoryPath)) {
			$scope.configForm.passwordStoreDirectoryPath.$setValidity('validity', false);
		} else {
			$scope.configForm.passwordStoreDirectoryPath.$setValidity('validity', true);
		}
	}, true);

	$scope.addConfig = async function () {
		if ($scope.configForm.$invalid) {
			return;
		}
		if($scope.config.passwordStoreDirectoryPath.endsWith('/')) {
			$scope.config.passwordStoreDirectoryPath = $scope.config.passwordStoreDirectoryPath.substring(0, $scope.config.passwordStoreDirectoryPath.length - 1);
		}
		store.set('config', $scope.config);
		ipcRenderer.send("add:config:done", $scope.config);
	}
}


angular.module('passAppConfig', ['ngMaterial', 'ngMessages'])
	.controller('configController', ConfigController);
