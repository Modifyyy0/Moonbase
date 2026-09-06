import websocket
import json

ws = websocket.create_connection("ws://localhost:8080/ws")

message = {
    "type": "send_message",
    "data": {
        "conversation_id": 12,
        "content": "hello everyone!"
    }
}

ws.send(json.dumps(message))

ws.close()