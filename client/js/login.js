import { rest } from "./rest.js";

const loginUsernameInput = document.getElementById("login-username-input")
const loginButton = document.getElementById("login-button")


loginButton.addEventListener("click", () => {
    const username = loginUsernameInput.value
    console.log(username)
    rest.login(username)
}) 