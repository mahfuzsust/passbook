const electron = require("electron");
const {ipcRenderer, remote} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

	let credential = {
        bookId: remote.getCurrentWindow().bookId,
        name: document.getElementById("credential_name").value,
        url: document.getElementById("credential_url").value,
        username: document.getElementById("credential_username").value,
        password: document.getElementById("credential_password").value
    };
	ipcRenderer.send("credential:add", credential);
});
