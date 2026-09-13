import { rest } from "./rest.js";

const loginUsernameInput = document.getElementById("login-username-input");
const loginButton = document.getElementById("login-button");

const handleLogin = async () => {
    const username = loginUsernameInput.value.trim();

    if (!username) {
        return;
    }

    try {
        await rest.login(username);
        window.location.href = "chat.html";
    } catch (error) {
        console.error("Login failed:", error);
        alert("Unable to log in right now.");
    }
};

loginButton.addEventListener("click", handleLogin);
loginUsernameInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
        handleLogin();
    }
});