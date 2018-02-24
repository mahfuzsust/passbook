const electron = require("electron");
const {ipcRenderer} = electron;

const ul = document.querySelector("#book_list");
const new_book = document.querySelector("#new_book");

document.getElementById("new_book").addEventListener("click", function(e){
	e.preventDefault();
	e.stopPropagation();

    ipcRenderer.send("click:bookadd", null);
});

ipcRenderer.on("book:add", function(e, item) {
	ul.className = "collection";
	let li = document.createElement("li");
	li.id = item._id;
	li.className = "collection-item";
	let text = document.createTextNode(item.name);
	li.appendChild(text);
	ul.appendChild(li);
});

ipcRenderer.on("book:list", function(e, books) {
	console.log('booklist test');
	
	ul.className = "collection";
	for (var i = 0; i < books.length; i++) 
	{
		var item = books[i];
		let li = document.createElement("li");
		li.id = item._id;
		li.className = "collection-item";
		let text = document.createTextNode(item.name);
		li.appendChild(text);
		ul.appendChild(li);
	}
});