#!/bin/bash
# prune
docker system prune -a
# Delete all containers
docker rm $(docker ps -a -q)
# Delete all images
docker rmi $(docker images -q)
