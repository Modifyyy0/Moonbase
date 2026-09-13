import websocket
import json
import threading
import time

ws = websocket.create_connection(
    "ws://localhost:8080/ws",
    cookie="session_token=abc123"
)

def receive_message():
    try:
        message = ws.recv()
        print("RECEIVED:", message)
    except Exception as e:
        print("Receiver stopped:", e)

receiver = threading.Thread(
    target=receive_message,
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
ws.send(json.dumps(message))

time.sleep(2)

ws.close()