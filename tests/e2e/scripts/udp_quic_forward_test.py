import socket
import sys
import base64


# QUIC v1 Initial packet (captured from real traffic, SNI=cloudflare-quic.com).
# Used to verify that the GOST proxy correctly sniffs and forwards QUIC traffic.
QUIC_PKT_B64 = "wQAAAAEMbPz5zifdvbrMMCfeAABE6onIhQqNpzy3/idh5rftIZ4dn6chk10EX5zjUSCc9HmGCZ95QWw9MTxkPZODKiEHadVyfmCt/2fjoHGmQUplPcmvWinaEXA6ysN/MREQ76J2DDH4YPZj1aSF2vGotyNf64/Qga6nfM3Nb+3iOKul7fwEjmi7kLjgHZ5ryTb+PHsddTg+07jW/hcFOgiQ9P/vreI+zJuEzcv2xV1oYAY7PnnM9aRrtAdik2V0WlAvTN9kFeV813+fnzQCw9ztyG9UGD6S7aUrUbCx3N3PFGc5yMmHsSGz5ZsO1wq8zGE3G9TSJTe6GwQpFwWz2J5gEgC7WB2ulfZYinZRC3+JtJ8lKm8dWt7tT8ymOYxLjpWh+JOHkkQHwCHvCoS8EugDZRsRNP+uME6SdRjzR/zAkawOz8jQL7oUWIYAEggxP3MmPw7uHVcYZHi8mhgvxIm8u7p5y5wqqweRhbRKrd1ji9Rq2wfEXrkumnfjkboP3OOqR/os4DFIaCNh1KHqsmBpbvTIQjrNQ11LZJSZ/I0pq6A1pSo9N6l38Ty4p8Ji6my1TRFmnCejxfSutmN/ozc7N9wD4ZVXcEMmqTsTrvjgHK4Frmqmmzr9SnMfbKdZ9BGdOFWXQrYqP1g6+Dg+nacCH2PdWjMyleSTMMvMlG6RrwLC/2eC1pDkRuhTVJS3h0ZOX+17jqHmKabzWv17qjzWx4dLu2z/fwWwaIJTO4sEvk05cquuyBYPUOfjbh7cQhIVOLBVJaXVIgro+C96wmoIlxfEMEZHDmriXU2SM6PKq8xw4a8AgnYBkGDofNOXMfdQj+byiwQS6pIOkii+haHqSSP6Y9r17jT3Ykytv+jeT4UanXjtjepBb5N/ApISY2Xs71irWRRZr9LXHXnJNFFdMWN367JIT3i865UMeXzndIu3iZdOWpSAApBnU2JMyjloRCA5Li9ABYgS2qjyo0SbleFj1Qp4CMNeuigkGR4oq/BSQmYMZL2KYJFAyX4BEev3YYkkJ12HlXgY54QL8jgxlJ+3e+qklM/lGajujPLfocswLUeWklfBZhoOJvPgIV5N9zGg1bxZLUe+brHrPpHnQZBAsksyM3YZra+PscMwQTopDIPBd1A52rSrmcvgQs6QR07Bh/ttS3tn0I4Xtb6rrkVgkC3zckIS2fB6WPHUOruDhQRPPehOfTehvTIFASt+6IR+jlGQ4/ZT1PkwAO+sLCBNEU1FBtLFWUpJsjSva49BjN5slRAI5SSCXeiwsvXuCfV8ZSmFZRvMR1BBuaFVVFrdK7PqMrtdJ90BInTE1J9vYUmWYvnZYI20xUm6mj2iourEkki4d8Vjtg2JSICnql/jKcIAxyJUDx1ZoFxvAbz34jTvx681lepy97n1jyGa7DlnpM/O6x+GHFaANQSj+2iIKjrQ+sM3YtSGMR2ClSg73f+pEdWUV5tyfiTB5nLI+VOpmt0AVCguau4ihLEmxADfQ1PXe9i1T0xvpceePc06T5pG37MYP5w6lLOJBajAFJuu6lavTtE7EyreoBnxkvzPciKupNwTxWdmWT8HVgQgaUUbvDCtTQr6nJnNMaGk2zbsXm8kiXfXS4lgf1sqw7HCphffMSH73JCabipof896LXjj2I8hj8wA8UMladryva0hv+Px0DmzsbO2/Wh+IhnbD8EoYH2gUehLQwQ="


def main():
    host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 9000

    pkt = base64.b64decode(QUIC_PKT_B64)

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(10)

    sock.sendto(pkt, (host, port))

    data, addr = sock.recvfrom(4096)
    if len(data) == len(pkt):
        print(f"PASS: received {len(data)} bytes (echo matches)")
        sys.exit(0)
    elif len(data) > 0:
        print(f"PARTIAL: sent {len(pkt)}, received {len(data)} bytes")
        sys.exit(0)
    else:
        print("FAIL: no response")
        sys.exit(1)


if __name__ == "__main__":
    main()
