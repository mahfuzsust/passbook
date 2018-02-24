const electron = require('electron');
const app = electron.app;
const BrowserWindow = electron.BrowserWindow;
const Menu = electron.Menu;
const ipcMain = electron.ipcMain; 

const path = require('path');
const url = require('url');
const db = require("./storage.js");

let mainWindow;
let addBookWindow;
let addLogin;

let mainMenuTemplate = [{
	label: "Add",
	submenu: [
	{label: "Book", click: createAddBookWindow},
	{label: "Login"}
	]
}];

if(process.platform == 'darwin') {
	mainMenuTemplate.unshift({});
}
if(process.env.NODE_ENV !== 'production') {
	mainMenuTemplate.push({
		label: "Developer Tools", 
		submenu: [{
			label: "Dev tools", 
			accelerator: process.platform == "darwin" ? "Command+I" : "Ctrl+I",
			click(item, focusedWindow) {
				focusedWindow.toggleDevTools();
			}
		}, {
			role: 'reload'
		}]
	});
}

function createWindow () {

	let menu = Menu.buildFromTemplate(mainMenuTemplate);
	Menu.setApplicationMenu(menu);

	mainWindow = new BrowserWindow({width: 800, height: 600})

	mainWindow.loadURL(url.format({
		pathname: path.join(__dirname, 'index.html'),
		protocol: 'file:',
		slashes: true
	}));
	//mainWindow.webContents.openDevTools()

	mainWindow.on('closed', function () {
		app.quit(); // should remove
		mainWindow = null
	});
}

app.on('ready', createWindow);

// Quit when all windows are closed.
app.on('window-all-closed', function () {
	if (process.platform !== 'darwin') {
		app.quit()
	}
});

app.on('activate', function () {
	if (mainWindow === null) {
		createWindow()
	}
});

function createAddBookWindow () {
	addBookWindow = new BrowserWindow({width: 400, height: 200, title: "Add book"})

	addBookWindow.loadURL(url.format({
		pathname: path.join(__dirname, 'addBook.html'),
		protocol: 'file:',
		slashes: true
	}));

	addBookWindow.on('close', function () {
		addBookWindow = null
	});
}

ipcMain.on("click:bookadd", function(e, item) {
	createAddBookWindow();
});

ipcMain.on("book:add", function (e, item) {
	mainWindow.webContents.send("book:add", item);
	addBookWindow.close();
	db.addBook(item);
});




