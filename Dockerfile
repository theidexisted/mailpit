FROM ubuntu:22.04 AS build
WORKDIR /app
RUN apt update 
RUN apt install  tar gcc git curl -y
RUN curl -sL https://deb.nodesource.com/setup_19.x |  bash -
RUN apt update 
RUN apt install  nodejs  -y
RUN curl -OL https://golang.org/dl/go1.18.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xvf go1.18.2.linux-amd64.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"
RUN git clone https://github.com/theidexisted/mailpit
RUN cd mailpit  \
	&& git checkout cgo_sqlite  \
	&&  ./build.sh

FROM ubuntu:22.04
COPY --from=build /app/mailpit/mailpit /usr/bin
RUN chmod +x  /usr/bin/mailpit
RUN ls  /usr/bin/
ENTRYPOINT ["/usr/bin/mailpit", "-d", "memory"]
