# Chitty-Chat
The instructions apply to opening the service in Visual Studio Code.
1. Clone the repository to your own machine.
2. In Visual Studio Code, open split terminal. The number of terminals is number of clients + one server.
3. In the server terminal, run: "go run server/server.go". Click allow on the pop-up.
4. In the client terminal(s), run: "go run client/client.go".
5. Provide a username. 
6. Then, type any messages up to 128 characters.
7. Join with as many clients as desired.
8. Lastly, you can leave the service, either by writing "leave" or by disconnecting from the server (shutting the client terminal).
