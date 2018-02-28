const electron = require("electron");
const {ipcRenderer, clipboard, remote, Notification, shell} = electron;
const db = require("../storage");
const crypt = require("../crypt");
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

	setBookNameText(item, li);
	setBookNameDeleteIcon(item, li);
	setBookNameEditIcon(item, li);

	ul.appendChild(li);
};

var addCredentialToTable = function(item, rowEl) {
	var newRow = rowEl || table.insertRow(0);
	var nameCell = rowEl ? rowEl.cells[0] : newRow.insertCell(0);
	var urlCell = rowEl ? rowEl.cells[1] : newRow.insertCell(1);
	var userNameCell = rowEl ? rowEl.cells[2] : newRow.insertCell(2);
	var passwordCell = rowEl ? rowEl.cells[3] : newRow.insertCell(3);
	var actionCell = rowEl ? rowEl.cells[4] : newRow.insertCell(4);

	urlCell.style.width = "5%";
	actionCell.style.width = "8%";
	nameCell.style.width = "30%";
	userNameCell.style.width = "30%";
	passwordCell.style.width = "27%";

	if(rowEl) {
		nameCell.innerHTML = "";
		urlCell.innerHTML = "";
		userNameCell.innerHTML = "";
		passwordCell.innerHTML = "";
	} else {
		newRow.className = "credential-row";
		setCredentialAction(item, newRow, actionCell);
	}	
	setCredentialUrl(item, urlCell);
	setCredentialPassword(item, passwordCell);

	setCredentialName(nameCell, item);
	setCredentialUsername(userNameCell, item);
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
ipcRenderer.on("credential:edited", function(e, item) {
	addCredentialToTable(item, selectedCredential);
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
function setCredentialUsername(userNameCell, item) {
	let textSpan = document.createElement("span");
	textSpan.className = "truncate-table";
	textSpan.appendChild(document.createTextNode(crypt.decrypt(item.username, userId)));
	userNameCell.appendChild(textSpan);
}

function setCredentialName(nameCell, item) {
	let textSpan = document.createElement("span");
	textSpan.className = "truncate-table";
	textSpan.appendChild(document.createTextNode(item.name));
	nameCell.appendChild(textSpan);
}

function setBookNameEditIcon(item, li) {
	let edit = document.createElement("span");
	edit.innerHTML = "<i class='fas fa-pencil-alt' style='float:right;'></i>";
	edit.addEventListener("click", function (e) {
		ipcRenderer.send("click:bookedit", item);
		selectedBookItem = li;
	});
	li.appendChild(edit);
}

function setBookNameDeleteIcon(item, li) {
	let deleteIcon = document.createElement("span");
	deleteIcon.innerHTML = "<i class='fas fa-trash-alt' style='float:right; margin-left:5px;'></i>";
	deleteIcon.addEventListener("click", function (e) {
		ipcRenderer.send("book:delete", item);
		selectedBookItem = li;
	});
	li.appendChild(deleteIcon);
}

function setBookNameText(item, li) {
	let textSpan = document.createElement("span");
	textSpan.className = "truncate";
	textSpan.appendChild(document.createTextNode(crypt.decrypt(item.name, userId)));
	li.appendChild(textSpan);
}

function setCredentialAction(item, newRow, actionCell) {
	let deleteIcon = document.createElement("span");
	deleteIcon.innerHTML = "<i class='fas fa-trash-alt' style='float:right; margin-left:5px;'></i>";
	deleteIcon.addEventListener("click", function (e) {
		ipcRenderer.send("credential:delete", item);
		selectedCredential = newRow;
	});
	let edit = document.createElement("span");
	edit.innerHTML = "<i class='fas fa-pencil-alt' style='float:right;'></i>";
	edit.addEventListener("click", function (e) {
		ipcRenderer.send("click:credential_edit", item);
		selectedCredential = newRow;
	});
	actionCell.appendChild(deleteIcon);
	actionCell.appendChild(edit);
}

function setCredentialPassword(item, passwordCell) {
	let textSpan = document.createElement("span");
	var passwordText = document.createTextNode(passUnicode);
	textSpan.appendChild(passwordText);

	var showEl = document.createElement("span");
	showEl.innerHTML = " <i class='fas fa-eye'></i>";
	showEl.addEventListener("click", function (e) {
		if (showEl.classList.length == 0) {
			showEl.classList.add("show");
			showEl.innerHTML = " <i class='fas fa-eye-slash'></i>";
			passwordText.nodeValue = crypt.decrypt(item.password, userId);
			textSpan.classList.toggle("truncate-table");
		}
		else {
			showEl.classList.remove("show");
			showEl.innerHTML = " <i class='fas fa-eye'></i>";
			passwordText.nodeValue = passUnicode;
			textSpan.classList.toggle("truncate-table");
		}
	});
	var copyEl = document.createElement("span");
	copyEl.innerHTML = " <i class='fas fa-copy'></i>";
	copyEl.addEventListener("click", function (e) {
		clipboard.writeText(crypt.decrypt(item.password, userId));
		new Notification('Title', {
			body: 'Lorem Ipsum Dolor Sit Amet'
		}).show();
	});

	passwordCell.appendChild(textSpan);
	passwordCell.appendChild(showEl);
	passwordCell.appendChild(copyEl);
}

function setCredentialUrl(item, urlCell) {
	var urlText = document.createElement("a");
	if(item.url) {
		urlText.appendChild(document.createTextNode("URL"));
		urlText.addEventListener("click", function (e) {
			e.preventDefault();
			shell.openExternal(item.url);
		});
	}
	
	urlCell.appendChild(urlText);
}

