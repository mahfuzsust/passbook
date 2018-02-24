const electron = require("electron");
const {ipcRenderer} = electron;

const ul = document.querySelector("#book_list");
const new_book = document.querySelector("#new_book");

document.getElementById("new_book").addEventListener("click", function(e){
	e.preventDefault();
	e.stopPropagation();
	console.log("test");
    ipcRenderer.send("click:bookadd", null);
});

ipcRenderer.on("book:add", function(e, item) {
	ul.className = "collection";
	let li = document.createElement("li");
	li.className = "collection-item";
	let text = document.createTextNode(item);
	li.appendChild(text);
	ul.appendChild(li);
})