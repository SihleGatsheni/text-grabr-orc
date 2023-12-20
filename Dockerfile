FROM golang:latest

WORKDIR /app

# Install tesseract-ocr and Leptonica and dependencies.
RUN apt-get update && apt-get install -y \
 libleptonica-dev \
 tesseract-ocr \
 libtesseract-dev

# Set necessary environment variables for CGO.
ENV CGO_CFLAGS="-I/usr/include/leptonica"
ENV CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu"

COPY . .

RUN go build -o orc_server

EXPOSE 8080

CMD ["./orc_server"]
