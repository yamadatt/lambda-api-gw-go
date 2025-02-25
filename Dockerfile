FROM public.ecr.aws/docker/library/golang:1.23 as build-image
WORKDIR /src
COPY go.mod go.sum main.go ./
RUN go build -o lambda-handler
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build-image /src/lambda-handler .
ENV DATABASE_URL="postgres://tesnavi:demodemo@koji-stag-db.cbaimfixitb4.ap-northeast-1.rds.amazonaws.com:5432/tesnavi?sslmode=require"
ENTRYPOINT ./lambda-handler
