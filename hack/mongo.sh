docker run -d \
     --name mongodb \
     --rm \
     -p 27017:27017 \
     -e MONGODB_USERNAME=vasu1124 \
     -e MONGODB_PASSWORD=some-important-password \
     -e MONGODB_ROOT_PASSWORD=some-important-password \
     -e MONGODB_DATABASE=guestbook \
     bitnami/mongodb:3.4.7-r0
