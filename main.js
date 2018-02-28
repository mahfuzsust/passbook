const electron = require('electron');
const bcrypt = require("bcryptjs");
const app = electron.app;
const BrowserWindow = electron.BrowserWindow;
const Menu = electron.Menu;
const ipcMain = electron.ipcMain; 

const path = require('path');
const url = require('url');
const db = require("./storage.js");
var crypt = require("./crypt");

function isDev() {
	return process.mainModule.filename.indexOf('app.asar') === -1;
}

let loggedInUser;

let mainWindow;
let addBookWindow;
let addLogin;
let loginWindow;
let addCredentialWindow;

function createWindow () {
	mainWindow = new BrowserWindow({
		width: 930, 
		height: 700,
		//resizable: false,
		fullscreen: false,
	});

	mainWindow.loadURL(url.format({
		pathname: path.join(__dirname, "app", 'login.html'),
		protocol: 'file:',
		slashes: true
	}));
	mainWindow.setMenu(null);
	
	
	mainWindow.on('closed', function () {
		//app.quit(); // should remove
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

function createAddBookWindow (book) {
	addBookWindow = new BrowserWindow({width: 400, height: 300, title: "Add book"})

	addBookWindow.loadURL(url.format({
		pathname: path.join(__dirname, "app", 'addBook.html'),
		protocol: 'file:',
		slashes: true
	}));
	
	if(book) {
		addBookWindow.book = book;
	} else {
		addBookWindow.userId = loggedInUser._id;
	}

	addBookWindow.on('close', function () {
		addBookWindow = null
	});
}

ipcMain.on("click:bookadd", function(e, item) {
	createAddBookWindow();
});

ipcMain.on("book:add", addBook());

ipcMain.on("click:bookedit", function(e, item) {
	createAddBookWindow(item);
});

ipcMain.on("book:edit", editBook());
ipcMain.on("book:delete", deleteBook());

ipcMain.on("credential:delete", deleteCredential());
ipcMain.on("login:check", login());

ipcMain.on("register:click", showRegistrationPage());
ipcMain.on("register", registerUser());

function registerUser() {
	return function (e, login) {
		db.addUser(login, function (err, user) {
			mainWindow.loadURL(url.format({
				pathname: path.join(__dirname, "app", 'login.html'),
				protocol: 'file:',
				slashes: true
			}));
		});
	};
}

function showRegistrationPage() {
	return function (e) {
		mainWindow.loadURL(url.format({
			pathname: path.join(__dirname, "app", 'register.html'),
			protocol: 'file:',
			slashes: true
		}));
	};
}

function addBook() {
	return function (e, item) {
		addBookWindow.close();
		item.name = crypt.encrypt(item.name, loggedInUser._id);
		db.addBook(item, function (err, newBook) {
			mainWindow.webContents.send("book:add", newBook);
		});
	};
}

function editBook() {
	return function (e, item) {
		addBookWindow.close();
		item.name = crypt.encrypt(item.name, loggedInUser._id);
		db.editBook(item, function (err, numAffected, affectedDocuments, upsert) {
			mainWindow.webContents.send("book:edited", affectedDocuments);
		});
	};
}

function login() {
	return function (e, login) {
		db.getUser(login.username, function (err, user) {
			if (user && bcrypt.compareSync(login.password, user.password)) {
				let menu = Menu.buildFromTemplate(mainMenuTemplate);
				Menu.setApplicationMenu(menu);

				mainWindow.loadURL(url.format({
					pathname: path.join(__dirname, "app", 'index.html'),
					protocol: 'file:',
					slashes: true
				}));
				loggedInUser = user;
				mainWindow.userId = loggedInUser._id;
			} else {
				mainWindow.webContents.send("login:failed");
			}
			
		});
	};
}

function deleteBook() {
	return function (e, item) {
		db.deleteBook(item._id, function (err, numRemoved) {
			mainWindow.webContents.send("book:deleted");
		});
	};
}

function deleteCredential() {
	return function (e, item) {
		db.deleteCredential(item._id, function (err, numRemoved) {
			mainWindow.webContents.send("credential:deleted");
		});
	};
}

function createAddCredentialWindow (bookId, credential) {
	addCredentialWindow = new BrowserWindow({width: 400, height: 600, title: "Add Credential"})

	addCredentialWindow.loadURL(url.format({
		pathname: path.join(__dirname, "app", 'addCredential.html'),
		protocol: 'file:',
		slashes: true
	}));
	addCredentialWindow.bookId = bookId;
	if(credential) {
		addCredentialWindow.credential = credential;
	}

	addCredentialWindow.on('close', function () {
		addCredentialWindow = null
	});
}

ipcMain.on("click:credentialadd", function(e, bookId) {
	createAddCredentialWindow(bookId);
});

ipcMain.on("click:credential_edit", function(e, item) {
	item["userId"] = loggedInUser._id;
	createAddCredentialWindow(item.bookId, item);
});

ipcMain.on("credential:edit", editCredential());

function editCredential() {
	return function (e, item) {
		addCredentialWindow.close();
		item.password = crypt.encrypt(item.password, loggedInUser._id);
		item.username = crypt.encrypt(item.username, loggedInUser._id);
		db.editCredential(item, function (err, numAffected, affectedDocuments, upsert) {
			mainWindow.webContents.send("credential:edited", affectedDocuments);
		});
	};
}


ipcMain.on("credential:add", function (e, item) {
	addCredentialWindow.close();
	item.password = crypt.encrypt(item.password, loggedInUser._id);
	item.username = crypt.encrypt(item.username, loggedInUser._id);
	db.addCredential(item, function (err, newCredential) {
		mainWindow.webContents.send("credential:added", newCredential);
	});
});

let mainMenuTemplate = [{
	label: "Settings",
	submenu: [
		{
			label: "Sign out",
			click() {
				mainWindow.setMenu(null);
				mainWindow.loadURL(url.format({
					pathname: path.join(__dirname, "app", 'login.html'),
					protocol: 'file:',
					slashes: true
				}));
				loggedInUser = null;
			}
		}
	]
}];

if(process.platform == 'darwin') {
	mainMenuTemplate.unshift({});
}
if(isDev()) {
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
};
