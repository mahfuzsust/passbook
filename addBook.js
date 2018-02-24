const electron = require("electron");
const {ipcRenderer} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

	let val = document.getElementById("book_name").value;
	ipcRenderer.send("book:add", val);
})