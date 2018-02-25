const electron = require("electron");
const {ipcRenderer, clipboard} = electron;
const db = require("./storage");
const crypt = require("./crypt");


const ul = document.querySelector("#book_list");
const new_book = document.querySelector("#new_book");
var table = document.querySelector("#credentials");
var selectedBookItem;

var isBookListEmpty = function() {
	return ul.children.length == 0;
};
var addBookToBooklist = function(item) {
	let li = document.createElement("li");
	li.id = item._id;
	li.className = "collection-item book-item";
	let text = document.createTextNode(item.name);
	li.appendChild(text);
	ul.appendChild(li);
};

var addCredentialToTable = function(item) {
	var newRow = table.insertRow(0);
	newRow.className = "credential-row";
	var nameCell = newRow.insertCell(0);
	var urlCell = newRow.insertCell(1);
	var userNameCell = newRow.insertCell(2);
	var passwordCell = newRow.insertCell(3);

	var urlText = "<a href='" + item.url + "'>URL</a>";

	var passwordText = document.createTextNode("\u2022\u2022\u2022\u2022\u2022\u2022\u2022");

	var showEl = document.createElement("a");
	showEl.innerHTML=" <i class='fas fa-eye'></i>";
	showEl.addEventListener("click",function(e) {
		console.log(passwordText);
		passwordText.nodeValue = item.password;
	});
	
	var copyEl = document.createElement("a");
	copyEl.innerHTML=" <i class='fas fa-copy'></i>";
	copyEl.addEventListener("click",function(e) {
		clipboard.writeText(item.password);
	});

	nameCell.appendChild(document.createTextNode(item.name));
	urlCell.innerHTML = urlText;
	userNameCell.appendChild(document.createTextNode(item.username));
	passwordCell.appendChild(passwordText);
	passwordCell.appendChild(showEl);
	passwordCell.appendChild(copyEl);
};

db.getAllBook(function(err, books) {
	if(isBookListEmpty() && books.length > 0) {
		ul.className += " collection";
	}
	for (var i = 0; i < books.length; i++) 
	{
		var item = books[i];
		addBookToBooklist(item);
	}
	ul.children[0].click();
});

ipcRenderer.on("book:add", function(e, item) {
	if(isBookListEmpty()) {
		ul.className += " collection";
	}
	addBookToBooklist(item);
});
ipcRenderer.on("credential:added", function(e, item) {
	addCredentialToTable(item);
});

var setCredentialByBookId = function(bookId) {
	db.getAllCredential(bookId, function(err, credentials) {
		table.innerHTML = "";
		for (var i = 0; i < credentials.length; i++) 
		{
			var item = credentials[i];
			addCredentialToTable(item);
		}
	});
};

ul.addEventListener("click",function(e) {
	//e.target.remove();
	if(e.target && e.target.nodeName == "LI") {
		console.log(e.target.id + " was clicked");
		if(selectedBookItem) {
			selectedBookItem.classList.toggle("book-item-clicked");
		}
		selectedBookItem = e.target;
		e.target.classList.toggle("book-item-clicked");
		setCredentialByBookId(selectedBookItem.id);
	}
});

document.getElementById("new_credential").addEventListener("click", function(e){
	e.preventDefault();
	e.stopPropagation();

    ipcRenderer.send("click:credentialadd", selectedBookItem.id);
});
document.getElementById("new_book").addEventListener("click", function(e){
	e.preventDefault();
	e.stopPropagation();

    ipcRenderer.send("click:bookadd", null);
});

function credentialClick(item) {
	console.log("click");
	console.log(item);
};

