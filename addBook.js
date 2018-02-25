const electron = require("electron");
const {ipcRenderer, remote} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

	let book = {
		name: document.getElementById("book_name").value,
		userId: remote.getCurrentWindow().userId
	};
	ipcRenderer.send("book:add", book);
});
