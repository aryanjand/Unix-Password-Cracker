FROM ubuntu:latest
RUN apt-get update && apt-get install -y golang tmux git curl
WORKDIR /app