# Copyright (c) 2021 6 River Systems
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

FROM debian:stable-slim
LABEL MAINTAINER="Matthew Gabeler-Lee <mgabeler-lee@6river.com>"

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/app/entrypoint.sh"]

RUN \
	apt-get update && \
	apt-get -y install dumb-init && \
	apt-get clean && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# TODO: don't use NODE_ENV for non-NodeJS code
ENV NODE_ENV=production
# Apps will need DATABASE_URL set externally to a useful value
# Apps will need PUBSUB_GCLOUD_PROJECT_ID set externally to a useful value

# This is the default port that would be used if this wasn't specified
ENV PORT=3000
EXPOSE 3000/tcp

RUN mkdir -p /app /data
VOLUME ["/data"]
WORKDIR /app

COPY .docker-deps/entrypoint.sh /app/

# this will default to the env var at build time
ARG BINARYNAME=${BINARYNAME}
# this makes it available at runtime
ENV BINARYNAME ${BINARYNAME}
# This is actually getting passed to the entrypoint as the _literal_ string (no
# interpolation), and the entrypoint recognizes this magic value and does the
# right thing with the runtime value of BINARYNAME
CMD ["/app/${BINARYNAME}"]

ARG TARGETARCH

# this uses the build-time arg/env
COPY bin/${BINARYNAME}-${TARGETARCH} /app/${BINARYNAME}
