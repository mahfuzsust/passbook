const electron = require("electron");
const {ipcRenderer} = electron;
const db = require("./storage");

const ul = document.querySelector("#book_list");
const new_book = document.querySelector("#new_book");
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
}

db.getAllBook(function(err, books) {
	if(isBookListEmpty()) {
		ul.className += " collection";
	}
	for (var i = 0; i < books.length; i++) 
	{
		var item = books[i];
		addBookToBooklist(item);
	}
});

document.getElementById("new_book").addEventListener("click", function(e){
	e.preventDefault();
	e.stopPropagation();

    ipcRenderer.send("click:bookadd", null);
});

ipcRenderer.on("book:add", function(e, item) {
	if(isBookListEmpty()) {
		ul.className += " collection";
	}

	addBookToBooklist(item);
});

ul.addEventListener("click",function(e) {
	//e.target.remove();

	if(e.target && e.target.nodeName == "LI") {
		console.log(e.target.id + " was clicked");
		if(selectedBookItem) {
			selectedBookItem.classList.toggle("book-item-clicked");
		}
		selectedBookItem = e.target;
		e.target.classList.toggle("book-item-clicked");
	}
});

