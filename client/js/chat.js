import { rest } from "./rest.js";
import { ChatWebSocket } from "./ws.js";
const messageInput = document.getElementById("message-input")
const convoContainer = document.getElementById("convo-container")
const createConversationButton = document.getElementById("new-convo-button")
const createConversationCloseButton = document.querySelector("#create-conversation-top-container button")
const popout = document.getElementById("popout")

const ws = new ChatWebSocket();
let current_conversation_id = 12

document.addEventListener("DOMContentLoaded", () => {
    ws.connect()
})

createConversationButton.addEventListener("click", () => {
    popout.style = "z-index: 1; opacity: 1"
})

createConversationCloseButton.addEventListener("click", () => {
    popout.style = "z-index: -1; opacity: 0"
})

messageInput.addEventListener("keydown", (event) => {
    if (event.key == "Enter" && !event.ctrlKey) {
        console.log("hmm")
        event.preventDefault()
        const content = messageInput.value.trim()

        if (content === "") {
            return
        }

        sendMessage("mango", "1:21", content)
        messageInput.value = ""
    }
    if (event.key == "Enter" && event.ctrlKey) {
        console.log("oh?")
        messageInput.value += "\n"
    }
})

const sendMessage = (user, time, content) => {
    convoContainer.appendChild(messageBox(user, time, content))
    
    const data = {
        "type": "send_message",
        "data": {
            "conversation_id": current_conversation_id,
            "content": content
        }
    }
    socket.send(JSON.stringify(data))
}

const messageBox = (user, time, content) => {
    const template = document.createElement("div")
    template.innerHTML = `
        <div class="message-box">
            <img class="message-pfp" src="assets/image.png" alt="">
            <div class="message-box-text">
                <p class="message-box-user"><span class="message-box-username"></span> <span
                        class="message-box-time"></span></p>
                <p class="message-content"></p>
            </div>
        </div>
    `
    const element = template.querySelector(".message-box");
    element.querySelector('.message-box-username').textContent = user
    element.querySelector('.message-box-time').textContent = time
    element.querySelector('.message-content').textContent = content

    return element
}

const dateLine = (date) => {
    const template = document.createElement('div')
    template.innerHTML = `
        <div class="date-line">
            <div></div>
            <p></p>
            <div></div>
        </div>
    `
    template.querySelector("p").textContent = date;
    return template.querySelector("date-line")
}

const memberButton = document.getElementById("info-bar-select-members")
const infoButton = document.getElementById("info-bar-select-info")
const infoBarMembers = document.getElementById("info-bar-members")
const infoBarInfo = document.getElementById("info-bar-info")

memberButton.addEventListener("click", () => {
    infoBarMembers.style = "display: block"
    infoBarInfo.style = "display: none"
    memberButton.classList.add("info-bar-select-selected")
    infoButton.classList.remove("info-bar-select-selected")
})

infoButton.addEventListener("click", () => {
    infoBarMembers.style = "display: none"
    infoBarInfo.style = "display: flex"
    infoButton.classList.add("info-bar-select-selected")
    memberButton.classList.remove("info-bar-select-selected")
})