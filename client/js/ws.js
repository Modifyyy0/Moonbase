// client/ws.js

export class ChatWebSocket {
    constructor() {
        this.socket = null;

        // Functions registered by the UI.
        this.handlers = {
            new_message: [],
            user_online: [],
            user_offline: [],
        };

        // Called when the connection opens/closes.
        this.onOpen = null;
        this.onClose = null;
        this.onError = null;
    }


    connect() {
        // Don't create another connection if one already exists.
        if (
            this.socket &&
            this.socket.readyState === WebSocket.OPEN
        ) {
            return;
        }

        // Use wss:// when the page is served over HTTPS.
        const protocol =
            window.location.protocol === "https:"
                ? "wss:"
                : "ws:";

        const host = "localhost:6767";
        const wsUrl = `${protocol}//${host}/ws`;

        this.socket = new WebSocket(wsUrl);

        
        this.socket.addEventListener("close", (event) => {
            console.log("WebSocket disconnected");

            if (this.onClose) {
                this.onClose(event);
            }
        });


        this.socket.addEventListener("open", () => {
            console.log("WebSocket connected");

            if (this.onOpen) {
                this.onOpen();
            }
        });


        this.socket.addEventListener("message", (event) => {
            this.handleMessage(event.data);
        });



        this.socket.addEventListener("error", (error) => {
            console.error("WebSocket error:", error);

            if (this.onError) {
                this.onError(error);
            }
        });
    }


    handleMessage(rawMessage) {
        let message;

        try {
            message = JSON.parse(rawMessage);
        } catch (error) {
            console.log("[WS]", rawMessage);
            return;
        }

        const { type, data } = message;

        // Ignore messages we don't recognize.
        if (!this.handlers[type]) {
            console.warn("Unknown WebSocket message type:", type);
            return;
        }

        // Notify every UI handler registered for this event.
        for (const handler of this.handlers[type]) {
            handler(data);
        }
    }


    sendMessage(conversationId, content) {
        if (
            !this.socket ||
            this.socket.readyState !== WebSocket.OPEN
        ) {
            throw new Error("WebSocket is not connected");
        }

        this.socket.send(JSON.stringify({
            type: "send_message",
            data: {
                conversation_id: conversationId,
                content: content,
            },
        }));
    }


    // Register a handler for a server event.
    on(type, handler) {
        if (!this.handlers[type]) {
            this.handlers[type] = [];
        }

        this.handlers[type].push(handler);

        // Return an unsubscribe function.
        return () => {
            this.handlers[type] =
                this.handlers[type].filter(h => h !== handler);
        };
    }


    disconnect() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
    }
}