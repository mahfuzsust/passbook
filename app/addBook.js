const electron = require("electron");
const crypt = require("../crypt");
const {ipcRenderer, remote} = electron;
var book = remote.getCurrentWindow().book;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();
	let name = document.getElementById("book_name").value;
	
	if(!name) {
		document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please enter name</span>";
		return;
	}

	if(book) {
		book.name = name;
		ipcRenderer.send("book:edit", book);
	} else {
		let elbook = {
			name: name,
			userId: remote.getCurrentWindow().userId
		};
		ipcRenderer.send("book:add", elbook);
	}
});

if(book) {
	document.getElementById("book_name").value = crypt.decrypt(book.name, book.userId);
}