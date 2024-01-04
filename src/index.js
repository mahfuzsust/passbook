const { app, BrowserWindow, ipcMain, Menu, shell } = require('electron');
const path = require('path');
const Store = require('electron-store');

const store = new Store();

if (require('electron-squirrel-startup')) {
	app.quit();
}

let addCredentialWindow;
let configlWindow;
let mainWindow;

const home = () => {
	if (store.get('config')) {
		createWindow();
	} else {
		createWindowForConfiguration();
	}
};

const createWindow = () => {
	mainWindow = new BrowserWindow({
		width: 960,
		height: 600,
		webPreferences: {
			devTools: !app.isPackaged,
			preload: path.join(__dirname, 'preload.js'),
			nodeIntegration: true,
			contextIsolation: false
		},
	});
	mainWindow.loadFile(path.join(__dirname, "app", 'index.html'));
};


function createAddCredentialWindow() {
	addCredentialWindow = new BrowserWindow({
		width: 530,
		height: 800,
		webPreferences: {
			devTools: !app.isPackaged,
			nodeIntegration: true,
			contextIsolation: false
		},
	});

	addCredentialWindow.loadFile(path.join(__dirname, "app", 'addCredential.html'));

	addCredentialWindow.on('close', function () {
		addCredentialWindow = null
	});
}

function createWindowForConfiguration() {
	configlWindow = new BrowserWindow({
		width: 530,
		height: 800,
		webPreferences: {
			devTools: !app.isPackaged,
			nodeIntegration: true,
			contextIsolation: false
		},
	});

	configlWindow.loadFile(path.join(__dirname, "app", 'config.html'));

	configlWindow.on('close', function () {
		configlWindow = null
	});
}


ipcMain.on("add:credential", function (e, item) {
	createAddCredentialWindow();
});

ipcMain.on("add:credential:done", function (e, item) {
	addCredentialWindow.close();
	mainWindow.reload();
});

ipcMain.on("add:config:done", function (e, item) {
	store.set('config', item);
	configlWindow.close();
	createWindow();
});

ipcMain.on("edit:credential", function (e, item) {
	createAddCredentialWindow();
	addCredentialWindow.webContents.on('did-finish-load', () => {
		addCredentialWindow.webContents.send("edit:credential:value", item);
	});
});

ipcMain.on("edit:credential:done", function (e, item) {
	addCredentialWindow.close();
	mainWindow.reload();
});

const template = [
	{
		label: 'File',
		submenu: [
			{
				label: 'Reload',
				click: () => {
					home();
				}
			},
			{
				label: 'Sync',
				click: () => {
					
				}
			},
			{
				label: 'Exit',
				click: () => {
					app.quit();
				}
			}
		]
	},
	{
		label: 'Edit',
		submenu: [
			{
				role: 'undo'
			},
			{
				role: 'redo'
			},
			{
				type: 'separator'
			},
			{
				role: 'cut'
			},
			{
				role: 'copy'
			},
			{
				role: 'paste'
			}
		]
	},
	{
		role: 'help',
		submenu: [
			{
				label: 'Learn More',
				click: () => {
					shell.openExternal('https://github.com/mahfuzsust/passbook');
				}
			}
		]
	}
]

const menu = Menu.buildFromTemplate(template)
Menu.setApplicationMenu(menu)

app.on('ready', home);
app.on('window-all-closed', () => {
	if (process.platform !== 'darwin') {
		app.quit();
	}
});

app.on('activate', () => {
	if (BrowserWindow.getAllWindows().length === 0) {
		home();
	}
});
