import websocket
import json

ws = websocket.create_connection(
    "ws://localhost:8080/ws",
    cookie="session_token=abc123"
)

message = {
    "type": "send_message",
    "data": {
        "conversation_id": 31,
        "content": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
    }
}

while True:
    ws.send(json.dumps(message))

    print(ws.recv())

ws.close()