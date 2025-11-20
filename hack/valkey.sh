#!/bin/bash
# Run Valkey locally for development

docker run -d \
     --name valkey \
     --rm \
     -p 6379:6379 \
     -e VALKEY_PASSWORD=some-important-password \
     valkey/valkey:8.1.4

echo "Valkey is running on localhost:6379"
echo "Password: some-important-password"
echo "To stop: docker stop valkey"
echo "To test: docker exec valkey valkey-cli -a some-important-password ping"
