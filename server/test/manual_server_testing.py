import websocket
import json
import threading
import time

ws = websocket.create_connection(
    "ws://localhost:8080/ws",
    cookie="session_token=abc123"
)

def receive_messages():
    while True:
        try:
            message = ws.recv()
            print("RECEIVED:", message)
        except Exception as e:
            print("Receiver stopped:", e)
            break

receiver = threading.Thread(
    target=receive_messages,
    daemon=True
)

receiver.start()

message = {
    "type": "send_message",
    "data": {
        "conversation_id": 1,
        "content": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
    }
}

while True:
    ws.send(json.dumps(message))
    print("SENT")

    time.sleep(2)