import requests

response = requests.get("http://localhost:8080/users")

print("Status:", response.status_code)
print("Headers:", response.headers)
print("Body:", response.text)