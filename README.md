# to proxy incoming ws requests to a target server

2 Approaches:
1. Upgrade an incoming http request to WS and then perform 2 way read/write of data between client and target server
2. (low level approach) Hijack the incoming http request and create a tcp connection to target server, and then copy data between the 2 tcp connections