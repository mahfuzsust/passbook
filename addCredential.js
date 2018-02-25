const electron = require("electron");
const {ipcRenderer, remote} = electron;
const passgen = require("./passgen");
const crypt = require("./crypt");
var credential = remote.getCurrentWindow().credential;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

	let newCredential = {
        bookId: remote.getCurrentWindow().bookId,
        name: document.getElementById("credential_name").value,
        url: document.getElementById("credential_url").value,
        username: document.getElementById("credential_username").value,
        password: document.getElementById("credential_password").value
    };
    if(credential) {
        newCredential["_id"] = credential._id;
        ipcRenderer.send("credential:edit", newCredential);
    } else {
        ipcRenderer.send("credential:add", newCredential);
    }
});

if(credential) {
    document.getElementById("credential_name").value = credential.name;
    document.getElementById("credential_url").value = credential.url;
    document.getElementById("credential_username").value = crypt.decrypt(credential.username, credential.userId);
    document.getElementById("credential_password").value = crypt.decrypt(credential.password, credential.userId);
}


document.getElementById("generate_password").addEventListener("click", function(e) {
    document.getElementById("credential_password").value = passgen.generatePassword()
});