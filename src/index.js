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
			preload: path.join(__dirname, 'preload.js'),
			nodeIntegration: true,
			contextIsolation: false
		}
	});

	if (!app.isPackaged) {
		mainWindow.webContents.openDevTools();
	}

	mainWindow.loadFile(path.join(__dirname, "app", 'index.html'));
};


function createAddCredentialWindow() {
	addCredentialWindow = new BrowserWindow({
		width: 530,
		height: 800,
		webPreferences: {
			nodeIntegration: true,
			contextIsolation: false
		}
	});
	
	if (!app.isPackaged) {
		addCredentialWindow.webContents.openDevTools();
	}
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
			nodeIntegration: true,
			contextIsolation: false
		}
	});

	if (!app.isPackaged) {
		configlWindow.webContents.openDevTools();
	}

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
	configlWindow.close();
	mainWindow.reload();
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
				accelerator: 'CmdOrCtrl+R',
				click: () => {
					mainWindow.reload();
				}
			},
			{
				label: 'Configuration',
				accelerator: 'CmdOrCtrl+P',
				click: () => {
					createWindowForConfiguration();
				}
			},
			{
				label: 'Sync',
				accelerator: 'CmdOrCtrl+S',
				click: () => {
					mainWindow.webContents.send("passbook:sync", true);
				}
			},
			{
				label: 'Exit',
				accelerator: 'CmdOrCtrl+Q',
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
				role: 'selectAll'
			},
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
