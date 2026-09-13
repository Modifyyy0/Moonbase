import { rest } from "./rest.js";
import { ChatWebSocket } from "./ws.js";

const elements = {
    messageInput: document.getElementById("message-input"),
    convoContainer: document.getElementById("convo-container"),
    newConversationButton: document.getElementById("new-convo-button"),
    inviteButton: document.getElementById("invite-button"),
    createConversationCloseButton: document.querySelector("#create-conversation-top-container button"),
    createConversationSearchInput: document.querySelector("#create-conversation-middle-container input"),
    createConversationNameInput: document.querySelector("#create-conversation-bottom-container input"),
    createConversationButton: document.querySelector("#create-conversation-bottom-container button"),
    inviteUserSearch: document.getElementById("create-conversation-user-search"),
    conversationList: document.getElementById("side-bar-list"),
    popout: document.getElementById("popout"),
    memberButton: document.getElementById("info-bar-select-members"),
    infoButton: document.getElementById("info-bar-select-info"),
    infoBarMembers: document.getElementById("info-bar-members"),
    infoBarInfo: document.getElementById("info-bar-info"),
    logoutButton: document.getElementById("logout-button"),
    leaveButton: document.getElementById("leave-button"),
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
    inviteSelections: [],
    inviteUsers: [],
};

const setPopoutVisibility = (isOpen) => {
    elements.popout.style.zIndex = isOpen ? "1" : "-1";
    elements.popout.style.opacity = isOpen ? "1" : "0";
};

const normalizeUser = (user) => ({
    id: Number(user?.id ?? user?.ID ?? 0),
    name: user?.name ?? user?.username ?? user?.Username ?? "Unknown",
});

const openPopout = async () => {
    state.inviteSelections = [];
    elements.createConversationNameInput.value = "";
    elements.createConversationSearchInput.value = "";
    await renderInviteUsers();
    setPopoutVisibility(true);
};

const closePopout = () => {
    state.inviteSelections = [];
    elements.createConversationSearchInput.value = "";
    elements.createConversationNameInput.value = "";
    setPopoutVisibility(false);
};

const formatTime = (date = new Date()) => {
    return new Date(date).toLocaleTimeString([], {
        timeZone: "UTC",
        hour: "2-digit",
        minute: "2-digit",
        // hour12: false
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
    const memberCount = Number(convo?.member_count ?? convo?.members?.length ?? 1);

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
            const username = message.sender_username ?? message.Username ?? "unknown";
            const content = message.content ?? message.Content ?? "";
            const sentAt = message.sent_time ?? message.SentAt ?? new Date();
            addMessageToView(username, content, sentAt);
            elements.convoContainer.scrollTo({
                top: elements.convoContainer.scrollHeight,
                behavior: "smooth"
            });
        });
    } catch (error) {
        console.error("Failed to load conversation messages:", error);
    }
};

const selectConversation = async (conversationId) => {
    const validId = Number(conversationId);
    state.activeConversationId = validId;
    if (!validId) {
        renderConversations([])
        populateConversationHeader({})
        return;
    }


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
            selectConversation(firstId);

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
    elements.convoContainer.scrollTop = elements.convoContainer.scrollHeight;
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


    console.log("Sending chat message for conversation:", state.activeConversationId, content);
    ws.sendMessage(state.activeConversationId, content);
};

const renderInviteUsers = async (query = "") => {
    try {
        const users = await rest.getUsers(query);
        state.inviteUsers = (Array.isArray(users) ? users : []).map(normalizeUser);

        elements.inviteUserSearch.innerHTML = "";

        if (!state.inviteUsers.length) {
            const emptyMessage = document.createElement("p");
            emptyMessage.textContent = "No users found";
            emptyMessage.style.color = "#4d4d4d";
            emptyMessage.style.fontSize = "20px";
            elements.inviteUserSearch.appendChild(emptyMessage);
            return;
        }

        state.inviteUsers.forEach((user) => {
            const row = document.createElement("div");
            row.className = "invite-user-row";

            const isSelected = state.inviteSelections.some((entry) => entry.id === user.id);
            const circle = document.createElement("span");
            circle.className = `invite-user-circle ${isSelected ? "selected" : ""}`;
            circle.textContent = isSelected ? "✓" : "";

            const label = document.createElement("span");
            label.textContent = user.name;

            row.append(circle, label);
            row.addEventListener("click", () => toggleInviteSelection(user));
            elements.inviteUserSearch.appendChild(row);
        });
    } catch (error) {
        console.error("Failed to load users:", error);
    }
};

const toggleInviteSelection = (user) => {
    const alreadySelected = state.inviteSelections.some((entry) => entry.id === user.id);

    if (alreadySelected) {
        state.inviteSelections = state.inviteSelections.filter((entry) => entry.id !== user.id);
    } else {
        state.inviteSelections.push(user);
    }

    renderInviteUsers(elements.createConversationSearchInput.value.trim());
};

const handleCreateConversation = async () => {
    const conversationName = elements.createConversationNameInput.value.trim();

    if (!conversationName) {
        return;
    }

    const invitedUsernames = state.inviteSelections.map((user) => user.name);

    closePopout();

    try {
        await rest.createConversation(conversationName, invitedUsernames);
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

const handleLeaveConversation = async () => {
    if (!state.activeConversationId) {
        return;
    }

    const confirmed = window.confirm("Leave this conversation?");
    if (!confirmed) {
        return;
    }

    try {
        await rest.leaveConversation(state.activeConversationId);
        await loadConversations();
    } catch (error) {
        console.error("Failed to leave conversation:", error);
    }
};

const bindEvents = () => {
    ws.on("new_message", handleIncomingMessage);
    ws.on("user_online", (data) => handlePresenceUpdate("user_online", data));
    ws.on("user_offline", (data) => handlePresenceUpdate("user_offline", data));

    elements.newConversationButton.addEventListener("click", openPopout);
    elements.createConversationCloseButton.addEventListener("click", closePopout);
    elements.createConversationButton.addEventListener("click", handleCreateConversation);
    elements.createConversationSearchInput.addEventListener("input", async (event) => {
        await renderInviteUsers(event.target.value.trim());
    });
    elements.messageInput.addEventListener("keydown", handleMessageInputKeydown);
    elements.memberButton.addEventListener("click", () => toggleInfoPanel("members"));
    elements.infoButton.addEventListener("click", () => toggleInfoPanel("info"));
    elements.inviteButton.addEventListener("click", openPopout);

    if (elements.logoutButton) {
        elements.logoutButton.addEventListener("click", logout);
    }

    if (elements.leaveButton) {
        elements.leaveButton.addEventListener("click", handleLeaveConversation);
    }
};

document.addEventListener("DOMContentLoaded", async () => {
    bindEvents();
    ws.connect();
    await loadConversations();
});
