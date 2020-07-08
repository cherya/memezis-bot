FROM golang
# All these steps will be cached
RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN echo "[url \"git@github.com:\"]\n\tinsteadOf = https://github.com/" >> /root/.gitconfig
RUN mkdir /root/.ssh && echo "StrictHostKeyChecking no " > /root/.ssh/config

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

RUN pwd
RUN ls -la

# Build the binary
RUN make build

CMD ["/app/bin/memezisbot", "--env=.production.env"]