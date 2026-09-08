import { rest } from "./rest.js";
const messageInput = document.getElementById("message-input")
const convoContainer = document.getElementById("convo-container")
const socket = new WebSocket("ws://localhost:8080/ws");

let current_conversation_id = 12

socket.addEventListener("open", () => {
    console.log("Connected to WebSocket server");
});

socket.addEventListener("message", (event) => {
    const payload = JSON.parse(event.data);
    console.log("Received:", payload);

    if (payload.type === "new_message") {
        // Sample payload
        // {
        //      "type": "new_message",
        //      "data": {
        //          "id": 205,
        //          "conversation_id": 12,
        //          "sender_id": 42,
        //          "sender_username": "mango",
        //          "content": "hello everyone!",
        //          "sent_time": "2026-09-02T10:17:05Z"
        //      }
        // }
        

    } else if (payload.type === "user_online") {

    } else if (payload.type === "user_offline") {

    }

});

socket.addEventListener("close", () => {
    rest.login("Mango")
    console.log("Disconnected");
});

socket.addEventListener("error", (error) => {
    console.error("WebSocket error:", error);
});


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