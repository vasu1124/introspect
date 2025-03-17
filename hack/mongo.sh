docker run -d \
     --name mongodb \
     --rm \
     -p 27017:27017 \
     -e MONGODB_ROOT_PASSWORD=some-important-password \
     bitnami/mongodb:4.4
