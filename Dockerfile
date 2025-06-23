FROM golang:1.23-alpine

RUN apk add --no-cache bash curl

RUN go install github.com/pressly/goose/v3/cmd/goose@latest
WORKDIR /app

COPY . .
RUN chmod +x /wait-for-it.sh
RUN go build -o main .

ENTRYPOINT ["sh", "-x", "-c", \
                       "/wait-for-it.sh postgres:5432 -t 15 && \
                        echo \"DB_SOURCE=$DB_SOURCE\" && \
                        goose -dir /app/db/migration postgres \"$DB_SOURCE\" up && \
                        ./main"]