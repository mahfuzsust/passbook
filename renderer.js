const electron = require("electron");
const {ipcRenderer, clipboard, remote, Notification, shell} = electron;
const db = require("./storage");
const crypt = require("./crypt");
const userId = remote.getCurrentWindow().userId;
const passUnicode = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
const ul = document.querySelector("#book_list");
const new_book = document.querySelector("#new_book");

var table = document.querySelector("#credentials");

var selectedBookItem;
var selectedCredential;

var isBookListEmpty = function() {
	return ul.children.length == 0;
};
var addBookToBooklist = function(item, liEl) {
	let li;
	if(liEl) {
		li = liEl;
		li.innerHTML = "";
	} else {
		li = document.createElement("li");
	}

	li.id = item._id;
	li.className = "collection-item book-item";
	let deleteIcon = document.createElement("span");
	deleteIcon.innerHTML = "<i class='fas fa-trash-alt' style='float:right; margin-left:5px;'></i>";
	deleteIcon.addEventListener("click", function(e){
		ipcRenderer.send("book:delete", item);
		selectedBookItem = li;
	});

	let edit = document.createElement("span");
	edit.innerHTML = "<i class='fas fa-pencil-alt' style='float:right;'></i>";
	edit.addEventListener("click", function(e){
		ipcRenderer.send("click:bookedit", item);
		selectedBookItem = li;
	});

	let text = document.createTextNode(crypt.decrypt(item.name, userId));
	li.appendChild(text);
	li.appendChild(deleteIcon);
	li.appendChild(edit);
	ul.appendChild(li);
};

var addCredentialToTable = function(item, rowEl) {
	var newRow;
	if(rowEl) {
		newRow = rowEl;
	} else {
		newRow = table.insertRow(0);
	}

	newRow.className = "credential-row";
	var nameCell = newRow.insertCell(0);
	var urlCell = newRow.insertCell(1);
	var userNameCell = newRow.insertCell(2);
	var passwordCell = newRow.insertCell(3);
	var actionCell = newRow.insertCell(4);

	var urlText = document.createElement("a");
	urlText.appendChild(document.createTextNode("URL"));
	urlText.addEventListener("click", function(e) {
		e.preventDefault();
		shell.openExternal(item.url);
	});

	var passwordText = document.createTextNode(passUnicode);

	var showEl = document.createElement("span");
	showEl.innerHTML=" <i class='fas fa-eye'></i>";
	showEl.addEventListener("click",function(e) {
		console.log(showEl.classList);
		if(showEl.classList.length == 0) {
			showEl.classList.add("show");
			showEl.innerHTML=" <i class='fas fa-eye-slash'></i>";
			passwordText.nodeValue = crypt.decrypt(item.password, userId);
		} else {
			showEl.classList.remove("show");
			showEl.innerHTML=" <i class='fas fa-eye'></i>";
			passwordText.nodeValue = passUnicode;
		}
	});
	
	var copyEl = document.createElement("span");
	copyEl.innerHTML=" <i class='fas fa-copy'></i>";
	copyEl.addEventListener("click",function(e) {
		clipboard.writeText(item.password);

		new Notification('Title', {
			body: 'Lorem Ipsum Dolor Sit Amet'
		}).show();

	});

	nameCell.appendChild(document.createTextNode(item.name));
	urlCell.appendChild(urlText);
	userNameCell.appendChild(document.createTextNode(crypt.decrypt(item.username, userId)));
	passwordCell.appendChild(passwordText);
	passwordCell.appendChild(showEl);
	passwordCell.appendChild(copyEl);

	let deleteIcon = document.createElement("span");
	deleteIcon.innerHTML = "<i class='fas fa-trash-alt' style='float:right; margin-left:5px;'></i>";
	deleteIcon.addEventListener("click", function(e){
		ipcRenderer.send("credential:delete", item);
		selectedCredential = newRow;
	});

	let edit = document.createElement("span");
	edit.innerHTML = "<i class='fas fa-pencil-alt' style='float:right;'></i>";
	edit.addEventListener("click", function(e){
		ipcRenderer.send("click:credential_edit", item);
		selectedCredential = newRow;
	});

	actionCell.appendChild(deleteIcon);
	actionCell.appendChild(edit);
};

db.getAllBook(userId, function(err, books) {
	if(isBookListEmpty() && books.length > 0) {
		document.getElementById("empty_book").style.display = "none";
		ul.style.display = "block";
		ul.className += " collection";

		for (var i = 0; i < books.length; i++) 
		{
			var item = books[i];
			addBookToBooklist(item);
		}
		ul.children[0].click();
	}
});

ipcRenderer.on("book:add", function(e, item) {
	if(isBookListEmpty()) {
		document.getElementById("empty_book").style.display = "none";
		ul.style.display = "block";
		ul.className += " collection";
	}
	addBookToBooklist(item);
});

ipcRenderer.on("book:edited", function(e, item) {
	addBookToBooklist(item, selectedBookItem);
});
ipcRenderer.on("book:deleted", function(e) {
	selectedBookItem.remove();
	if(isBookListEmpty()) {
		document.getElementById("empty_book").style.display = "block";
		ul.style.display = "none";
	}
});
ipcRenderer.on("credential:deleted", function(e) {
	selectedCredential.remove();
	if(document.getElementById("credentials").children.length == 0) {
		document.getElementById("empty_credential").style.display = "block";
		document.getElementById("credential_table").style.display = "none";
	}
});

ipcRenderer.on("credential:added", function(e, item) {
	if(document.getElementById("credentials").children.length == 0) {
		document.getElementById("empty_credential").style.display = "none";
		document.getElementById("credential_table").style.display = "block";
	}
	addCredentialToTable(item);
});

var setCredentialByBookId = function(bookId) {
	db.getAllCredential(bookId, function(err, credentials) {
		if(credentials.length > 0) {
			document.getElementById("empty_credential").style.display = "none";
			document.getElementById("credential_table").style.display = "block";

			table.innerHTML = "";
			for (var i = 0; i < credentials.length; i++) 
			{
				var item = credentials[i];
				addCredentialToTable(item);
			}
		} else {
			document.getElementById("empty_credential").style.display = "block";
			document.getElementById("credential_table").style.display = "none";
			document.getElementById("credentials").innerHTML = "";
		}
		
	});
};

ul.addEventListener("click",function(e) {
	//e.target.remove();
	if(e.target && e.target.nodeName == "LI") {
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
