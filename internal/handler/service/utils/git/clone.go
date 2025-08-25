package git

// This works flow is:
// 1. User sends a request to create a service with git config
// 2. We try to connect to the user and try to excute the git clone to /data/starker/services/{serviceID}
// 3. We try to get the docker compose from the service (we only support docker compose for now) but we do allow user to build the image
// 4. If everything success it should return the docker compose file
// 5. And after it successfull clone the project it should send the compose back to the service
// 6. If any step fails it should return the error to the user
// 7. it must use chan to send the data in real-time
