import socket


sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("0.0.0.0", 5679))

while True:
    data, addr = sock.recvfrom(2048)
    sock.sendto(data, addr)
