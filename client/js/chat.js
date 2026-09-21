import { rest } from "./rest.js";
import { ChatWebSocket } from "./ws.js";

const elements = {
    messageInput: document.getElementById("message-input"),
    convoContainer: document.getElementById("convo-container"),
    newConversationButton: document.getElementById("new-convo-button"),
    createConversationCloseButton: document.querySelector("#create-conversation-top-container button"),
    createConversationNameInput: document.querySelector("#create-conversation-bottom-container input"),
    createConversationButton: document.querySelector("#create-conversation-bottom-container button"),
    conversationList: document.getElementById("side-bar-list"),
    popout: document.getElementById("popout"),
    memberButton: document.getElementById("info-bar-select-members"),
    infoButton: document.getElementById("info-bar-select-info"),
    infoBarMembers: document.getElementById("info-bar-members"),
    infoBarInfo: document.getElementById("info-bar-info"),
    logoutButton: document.getElementById("logout-button"),
    convoHeadName: document.getElementById("convo-head-name"),
    convoHeadOnline: document.getElementById("convo-head-online"),
    convoHeadMembers: document.getElementById("convo-head-members"),
    infoBarOnline: document.getElementById("info-bar-online"),
    infoBarOffline: document.getElementById("info-bar-offline"),
    infoBarConvoName: document.getElementById("info-bar-convo-name"),
};

const ws = new ChatWebSocket();
const state = {
    conversations: [],
    activeConversationId: 19,
    currentConversation: null,
};

const setPopoutVisibility = (isOpen) => {
    elements.popout.style.zIndex = isOpen ? "1" : "-1";
    elements.popout.style.opacity = isOpen ? "1" : "0";
};

const openPopout = () => setPopoutVisibility(true);
const closePopout = () => setPopoutVisibility(false);

const formatTime = (date = new Date()) => {
    return new Date(date).toLocaleTimeString([], {
        hour: "numeric",
        minute: "2-digit",
    });
}

const createMessageBox = (user, time, content) => {
    const template = document.createElement("div");
    template.innerHTML = `
        <div class="message-box">
            <img class="message-pfp" src="assets/image.png" alt="">
            <div class="message-box-text">
                <p class="message-box-user">
                    <span class="message-box-username"></span>
                    <span class="message-box-time"></span>
                </p>
                <p class="message-content"></p>
            </div>
        </div>
    `;

    const element = template.querySelector(".message-box");
    element.querySelector(".message-box-username").textContent = user;
    element.querySelector(".message-box-time").textContent = time;
    element.querySelector(".message-content").textContent = content;

    return element;
};

const createSidebarConversation = (convo, { selected = false } = {}) => {
    const template = document.createElement("div");
    template.innerHTML = `
        <div class="side-bar-convo">
            <img class="side-bar-convo-pfp" src="assets/images/placeholder.png" alt="">
            <div class="side-bar-convo-text">
                <p class="side-bar-convo-name"></p>
                <p class="side-bar-convo-members"></p>
            </div>
        </div>
    `;

    const conversation = template.querySelector(".side-bar-convo");
    const name = convo?.name ?? convo?.Name ?? "Conversation";
    const memberCount = Number(convo?.memberCount ?? convo?.members?.length ?? 1);

    conversation.querySelector(".side-bar-convo-name").textContent = name;
    conversation.querySelector(".side-bar-convo-members").textContent = `${memberCount} members`;

    if (selected) {
        conversation.classList.add("active");
    }

    return conversation;
};

const renderConversations = (conversations = []) => {
    elements.conversationList.innerHTML = "";
    state.conversations = conversations;

    conversations.forEach((convo) => {
        const convoId = Number(convo?.id ?? convo?.ID ?? convo?.conversation_id);
        const selected = convoId === state.activeConversationId;
        const item = createSidebarConversation(convo, { selected });

        if (convoId) {
            item.addEventListener("click", () => selectConversation(convoId));
        }

        elements.conversationList.appendChild(item);
    });
};

const renderMemberList = (members = []) => {
    const existingCards = elements.infoBarMembers.querySelectorAll(".info-bar-user");
    existingCards.forEach((card) => card.remove());

    const onlineMembers = members.filter((member) => member.online);
    const offlineMembers = members.filter((member) => !member.online);
    console.log(onlineMembers)
    console.log(offlineMembers)

    elements.infoBarOnline.textContent = `Online (${onlineMembers.length})`;
    elements.infoBarOffline.textContent = `Offline (${offlineMembers.length})`;

    const onlineSection = document.createElement("div");
    onlineSection.className = "info-bar-member-section";

    const offlineSection = document.createElement("div");
    offlineSection.className = "info-bar-member-section";

    onlineMembers.forEach((member) => {
        const card = document.createElement("div");
        card.className = "info-bar-user";

        card.innerHTML = `
            <img class="info-bar-pfp" src="assets/images/placeholder.png" alt="">
            <p class="info-bar-username"></p>
        `;

        card.querySelector(".info-bar-username").textContent = member.username ?? member.Username ?? "Unknown";
        onlineSection.appendChild(card);
    });

    offlineMembers.forEach((member) => {
        const card = document.createElement("div");
        card.className = "info-bar-user";

        card.innerHTML = `
            <img class="info-bar-pfp" src="assets/images/placeholder.png" alt="">
            <p class="info-bar-username"></p>
        `;

        card.querySelector(".info-bar-username").textContent = member.username ?? member.Username ?? "Unknown";
        offlineSection.appendChild(card);
    });

    elements.infoBarOnline.after(onlineSection);
    elements.infoBarOffline.after(offlineSection);
};

const populateConversationHeader = (conversation = {}) => {
    state.currentConversation = conversation;

    const title = conversation.name ?? conversation.Name ?? "Conversation";
    const memberCount = Number(conversation.memberCount ?? conversation.members?.length ?? 0);
    const onlineCount = Array.isArray(conversation.members)
        ? conversation.members.filter((member) => member.online).length
        : 0;

    elements.convoHeadName.textContent = title;
    elements.convoHeadMembers.textContent = `${memberCount} members`;
    elements.convoHeadOnline.textContent = `${onlineCount} online`;
    elements.infoBarConvoName.textContent = title;
    renderMemberList(Array.isArray(conversation.members) ? conversation.members : []);
};

const loadConversationMessages = async (conversationId) => {
    try {
        const messages = await rest.getMessages(conversationId);
        elements.convoContainer.innerHTML = "";

        (messages || []).forEach((message) => {
            const username = message.sender_username ?? message.SenderUsername ?? "unknown";
            const content = message.content ?? message.Content ?? "";
            const sentAt = message.sent_time ?? message.SentAt ?? new Date();
            addMessageToView(username, content, sentAt);
        });
    } catch (error) {
        console.error("Failed to load conversation messages:", error);
    }
};

const selectConversation = async (conversationId) => {
    const validId = Number(conversationId);
    if (!validId) {
        return;
    }

    state.activeConversationId = validId;

    try {
        const conversation = await rest.getConversation(validId);
        populateConversationHeader(conversation || {});
        await loadConversationMessages(validId);
        renderConversations(state.conversations);
    } catch (error) {
        console.error("Failed to load selected conversation:", error);
    }
};

const loadConversations = async () => {
    try {
        const convos = await rest.getConversations();
        const conversationList = Array.isArray(convos) ? convos : [];
        renderConversations(conversationList);

        if (conversationList.length > 0) {
            const firstId = Number(conversationList[0].id ?? conversationList[0].ID ?? conversationList[0].conversation_id);
            if (firstId) {
                await selectConversation(firstId);
            }
        }
    } catch (error) {
        console.error("Failed to load conversations:", error);
    }
};

const toggleInfoPanel = (selectedTab) => {
    const isMembersSelected = selectedTab === "members";

    elements.infoBarMembers.style.display = isMembersSelected ? "block" : "none";
    elements.infoBarInfo.style.display = isMembersSelected ? "none" : "flex";

    elements.memberButton.classList.toggle("info-bar-select-selected", isMembersSelected);
    elements.infoButton.classList.toggle("info-bar-select-selected", !isMembersSelected);
};

const addMessageToView = (user, content, sentAt = new Date()) => {
    const time = formatTime(sentAt);
    elements.convoContainer.appendChild(createMessageBox(user, time, content));
};

const handleIncomingMessage = (message) => {
    if (!message || Number(message.conversation_id) !== state.activeConversationId) {
        return;
    }

    addMessageToView(message.sender_username || "unknown", message.content, message.sent_time || new Date());
};

const handlePresenceUpdate = (type, data) => {
    if (!data || Number(data.conversation_id) !== state.activeConversationId) {
        return;
    }

    if (!state.currentConversation || !Array.isArray(state.currentConversation.members)) {
        return;
    }

    const member = state.currentConversation.members.find((entry) =>
        Number(entry.id) === Number(data.user_id) || entry.username === data.username
    );

    if (!member) {
        return;
    }

    member.online = type === "user_online";
    populateConversationHeader(state.currentConversation);
};

const sendMessage = (content) => {
    if (!state.activeConversationId) {
        console.warn("No active conversation selected.");
        return;
    }

    // addMessageToView("mango", content);

    console.log("Sending chat message for conversation:", state.activeConversationId, content);
    ws.sendMessage(state.activeConversationId, content);
};

const handleCreateConversation = async () => {
    const conversationName = elements.createConversationNameInput.value.trim();

    if (!conversationName) {
        return;
    }

    closePopout();

    try {
        await rest.createConversation(conversationName, ["mango"]);
        await loadConversations();
    } catch (error) {
        console.error("Failed to create conversation:", error);
    }
};

const handleMessageInputKeydown = (event) => {
    if (event.key === "Enter" && !event.ctrlKey) {
        event.preventDefault();

        const content = elements.messageInput.value.trim();
        if (!content) {
            return;
        }

        sendMessage(content);
        elements.messageInput.value = "";
    }

    if (event.key === "Enter" && event.ctrlKey) {
        elements.messageInput.value += "\n";
    }
};

const logout = async () => {
    try {
        await rest.logout();
        window.location.href = "login.html";
    } catch (error) {
        console.error("Logout failed:", error);
        window.location.href = "login.html";
    }
};

const bindEvents = () => {
    ws.on("new_message", handleIncomingMessage);
    ws.on("user_online", (data) => handlePresenceUpdate("user_online", data));
    ws.on("user_offline", (data) => handlePresenceUpdate("user_offline", data));

    elements.newConversationButton.addEventListener("click", openPopout);
    elements.createConversationCloseButton.addEventListener("click", closePopout);
    elements.createConversationButton.addEventListener("click", handleCreateConversation);
    elements.messageInput.addEventListener("keydown", handleMessageInputKeydown);
    elements.memberButton.addEventListener("click", () => toggleInfoPanel("members"));
    elements.infoButton.addEventListener("click", () => toggleInfoPanel("info"));

    if (elements.logoutButton) {
        elements.logoutButton.addEventListener("click", logout);
    }
};

document.addEventListener("DOMContentLoaded", async () => {
    bindEvents();
    ws.connect();
    await loadConversations();
    await selectConversation(19);

});
