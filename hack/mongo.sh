docker run -d \
     --name mongodb \
     --rm \
     -p 27017:27017 \
     -e MONGO_INITDB_ROOT_PASSWORD=some-important-password \
     mongo:8.0.16
