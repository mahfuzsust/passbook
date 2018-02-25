const electron = require("electron");
const {ipcRenderer} = electron;

let form = document.querySelector("form");
form.addEventListener("submit", function(e) {
	e.preventDefault();
	e.stopPropagation();

    let username = document.getElementById("username").value;
    let password = document.getElementById("password").value;
    let repassword = document.getElementById("repassword").value;

    if(password && repassword && password === repassword && username) {
        ipcRenderer.send("register", {username: username, password:password});
    } else {
        if(!username) {
            document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please enter username</span>";
        }
        else if(!password) {
            document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please enter password</span>";
        }
        // else if(password.match(/^(?=.*[0-9])(?=.*[a-z])(?=.*[A-Z])([a-zA-Z0-9]{8,})$/)) {
        //     document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Password should be 8 characters long and 1 lowercase letter, 1 uppercase letter and 1 number</span>";
        // }
        else if(!repassword) {
            document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Please re-type password</span>";
        }
        else if(password !== repassword) {
            document.getElementById("errorMessage").innerHTML = "<span style='color: red'>Password didn't match</span>";
        }
    } 
});
