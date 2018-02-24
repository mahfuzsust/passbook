const electron = require("electron");
const {ipcRenderer} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

    let username = document.getElementById("username").value;
    let password = document.getElementById("password").value;
    ipcRenderer.send("login:check", {username: username, password:password});
});
