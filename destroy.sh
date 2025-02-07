#!/bin/bash
docker rm -vf $(docker ps -aq --filter "label=1m-client") 2>/dev/null
