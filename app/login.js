const electron = require("electron");
const {ipcRenderer} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

    let username = document.getElementById("username").value;
    let password = document.getElementById("password").value;

    if(!username) {
		document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please enter username</span>";
		return;
    }
    if(!password) {
		document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please enter password</span>";
		return;
	}

    ipcRenderer.send("login:check", {username: username, password:password});
});
document.getElementById("register").addEventListener("click", function(e) {
    ipcRenderer.send("register:click");
});
ipcRenderer.on("login:failed", function(e) {
	document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Login failed</span>";
});