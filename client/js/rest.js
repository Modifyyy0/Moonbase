const API_BASE = "/api";

async function request(path, options = {}) {
    const response = await fetch(`${API_BASE}${path}`, {
        ...options,

        headers: {
            "Content-Type": "application/json",
            ...options.headers,
        },

        // Important: allows the browser to send cookies.
        credentials: "include",
    });

    // Try to parse JSON, even for error responses.
    const data = await response.json().catch(() => null);

    if (!response.ok) {
        throw new Error(
            data?.message || `Request failed (${response.status})`
        );
    }

    return data;
}


export const rest = {

    // POST /api/login
    async login(name) {
        return request("/login", {
            method: "POST",
            body: JSON.stringify({ name }),
        });
    },


    // POST /api/logout
    async logout() {
        return request("/logout", {
            method: "POST",
        });
    },


    // GET /api/me
    async getMe() {
        return request("/me");
    },


    // GET /api/conversations
    async getConversations() {
        return request("/conversations");
    },


    // POST /api/conversations
    async createConversation(Name, members) {
        return request("/conversations", {
            method: "POST",
            body: JSON.stringify({ Name, members }),
        });
    },


    // GET /api/conversations/{conversation_id}
    async getConversation(conversationId) {
        return request(`/conversations/${conversationId}`);
    },


    // GET /api/users?q=...
    async getUsers(query = "") {
        const params = query ? `?q=${encodeURIComponent(query)}` : "";
        return request(`/users${params}`);
    },


    // DELETE /api/conversations/{conversation_id}/members/me
    async leaveConversation(conversationId) {
        return request(
            `/conversations/${conversationId}/members/me`,
            {
                method: "DELETE",
            }
        );
    },


    // GET /api/conversations/{conversation_id}/messages
    async getMessages(conversationId) {
        return request(
            `/conversations/${conversationId}/messages`
        );
    },


    // POST /api/join/{conversation_id}
    async joinConversation(conversationId) {
        return request(`/join/${conversationId}`, {
            method: "POST",
        });
    },

};