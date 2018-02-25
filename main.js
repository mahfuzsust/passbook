const electron = require('electron');
const app = electron.app;
const BrowserWindow = electron.BrowserWindow;
const Menu = electron.Menu;
const ipcMain = electron.ipcMain; 

const path = require('path');
const url = require('url');
const db = require("./storage.js");

let loggedInUser;

let mainWindow;
let addBookWindow;
let addLogin;
let loginWindow;
let addCredentialWindow;

function createWindow () {

	let menu = Menu.buildFromTemplate(mainMenuTemplate);
	Menu.setApplicationMenu(menu);

	mainWindow = new BrowserWindow({
		width: 800, 
		height: 600,
		//resizable: false,
		fullscreen: false,
	});

	mainWindow.loadURL(url.format({
		pathname: path.join(__dirname, 'login.html'),
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
	addBookWindow.userId = loggedInUser._id;

	addBookWindow.on('close', function () {
		addBookWindow = null
	});
}

ipcMain.on("click:bookadd", function(e, item) {
	createAddBookWindow();
});

ipcMain.on("book:add", function (e, item) {
	addBookWindow.close();
	db.addBook(item, function (err, newBook) {
		mainWindow.webContents.send("book:add", newBook);
	});
});

ipcMain.on("login:check", function (e, login) {
	const bcrypt = require("bcryptjs");
	
    let dbhash = db.getUser(login.username, function(err, user) {
		if(bcrypt.compareSync(login.password, user.password)) {
			mainWindow.loadURL(url.format({
				pathname: path.join(__dirname, 'index.html'),
				protocol: 'file:',
				slashes: true
			}));
			loggedInUser = user;
			mainWindow.userId = loggedInUser._id;
        };
    });	
});

function createAddCredentialWindow (bookId) {
	addCredentialWindow = new BrowserWindow({width: 400, height: 600, title: "Add Credential"})

	addCredentialWindow.loadURL(url.format({
		pathname: path.join(__dirname, 'addCredential.html'),
		protocol: 'file:',
		slashes: true
	}));
	addCredentialWindow.bookId = bookId;

	addCredentialWindow.on('close', function () {
		addCredentialWindow = null
	});
}

ipcMain.on("click:credentialadd", function(e, bookId) {
	createAddCredentialWindow(bookId);
});
ipcMain.on("credential:add", function (e, item) {
	addCredentialWindow.close();
	db.addCredential(item, function (err, newCredential) {
		mainWindow.webContents.send("credential:added", newCredential);
	});
});

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