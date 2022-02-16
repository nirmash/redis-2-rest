# The base go-image
FROM golang:1.17-alpine
 
# Create a directory for the app
RUN mkdir /app
 
# Copy all files from the current directory to the app directory
COPY . /app
 
# Set working directory
WORKDIR /app

ENV OS_PORT=80
ENV REDIS_IP=redis
ENV REDID_PORT=6379

EXPOSE 80
 
# Run command as described:
# go build will build an executable file named server in the current directory
RUN go build -o server . 
 
# Run the server executable
CMD [ "/app/server" ]