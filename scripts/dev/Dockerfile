FROM ubuntu:focal-20230605

ENV DEBIAN_FRONTEND noninteractive
ENV ACCEPT_EULA y
ENV NODE_VERSION 18.17.0
ENV POSTGREST_VERSION 11.2.2

RUN ARCH= && dpkgArch="$(dpkg --print-architecture)" \
  && case "${dpkgArch##*-}" in \
  amd64) ARCH='x64';; \
  arm64) ARCH='arm64';; \
  i386) ARCH='x86';; \
  *) echo "unsupported architecture"; exit 1 ;; \
  esac \
  && set -ex \
  # libatomic1 for arm
  && apt-get update && apt-get install -y \
  ca-certificates \
  curl \
  wget \
  gnupg \
  dirmngr \
  xz-utils \
  libatomic1 \
  --no-install-recommends \
  && rm -rf /var/lib/apt/lists/* \
  && for key in \
  4ED778F539E3634C779C87C6D7062848A1AB005C \
  141F07595B7B3FFE74309A937405533BE57C7D57 \
  74F12602B6F1C4E913FAA37AD3A89613643B6201 \
  DD792F5973C6DE52C432CBDAC77ABFA00DDBF2B7 \
  61FC681DFB92A079F1685E77973F295594EC4689 \
  8FCCA13FEF1D0C2E91008E09770F7A9A5AE15600 \
  C4F0DFFF4E8C1A8236409D08E73BC641CC11F4C8 \
  890C08DB8579162FEE0DF9DB8BEAB4DFCF555EF4 \
  C82FA3AE1CBEDC6BE46B9360C43CEC45C17AB93C \
  108F52B48DB57BB0CC439B2997B01419BD92F80A \
  ; do \
  gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys "$key" || \
  gpg --batch --keyserver keyserver.ubuntu.com --recv-keys "$key" ; \
  done \
  && curl -fsSLO --compressed "https://nodejs.org/dist/v$NODE_VERSION/node-v$NODE_VERSION-linux-$ARCH.tar.xz" \
  && curl -fsSLO --compressed "https://nodejs.org/dist/v$NODE_VERSION/SHASUMS256.txt.asc" \
  && gpg --batch --decrypt --output SHASUMS256.txt SHASUMS256.txt.asc \
  && grep " node-v$NODE_VERSION-linux-$ARCH.tar.xz\$" SHASUMS256.txt | sha256sum -c - \
  && tar -xJf "node-v$NODE_VERSION-linux-$ARCH.tar.xz" -C /usr/local --strip-components=1 --no-same-owner \
  && rm "node-v$NODE_VERSION-linux-$ARCH.tar.xz" SHASUMS256.txt.asc SHASUMS256.txt \
  && apt-mark auto '.*' > /dev/null \
  && find /usr/local -type f -executable -exec ldd '{}' ';' \
  | awk '/=>/ { so = $(NF-1); if (index(so, "/usr/local/") == 1) { next }; gsub("^/(usr/)?", "", so); print so }' \
  | sort -u \
  | xargs -r dpkg-query --search \
  | cut -d: -f1 \
  | sort -u \
  | xargs -r apt-mark manual \
  && apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false \
  && ln -s /usr/local/bin/node /usr/local/bin/nodejs \
  # smoke tests
  && node --version \
  && npm --version

RUN mkdir -p /app && \
  mkdir -p /opt/hoop/sessions && \
  apt-get update -y && \
  apt-get install -y \
  net-tools \
  iproute2 \
  xz-utils \
  locales \
  tini \
  jq \
  openssh-client \
  openssh-server \
  sudo \
  apt-utils \
  procps \
  unzip \
  curl \
  gnupg \
  gettext-base \
  gnupg2 \
  libaio1 \
  wget

# # SSH Server configuration for testing
RUN echo 'root:1a2b3c4d' | chpasswd && \
  mkdir -p /run/sshd && \
  mkdir -p /root/.ssh && \
  ssh-keygen -A

RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ focal-pgdg main" | tee /etc/apt/sources.list.d/pgdg.list && \
  echo "deb [arch=amd64,arm64] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/5.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-5.0.list && \
  curl -sL https://packages.microsoft.com/config/ubuntu/20.04/prod.list | tee /etc/apt/sources.list.d/msprod.list && \
  curl -sL https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
  curl -sL https://www.mongodb.org/static/pgp/server-5.0.asc | apt-key add - && \
  curl -sL https://packages.microsoft.com/keys/microsoft.asc | apt-key add -

RUN apt-get update -y && \
  apt-get install -y \
  openjdk-11-jre \
  default-mysql-client \
  postgresql-client-15 \
  mongodb-mongosh mongodb-org-tools mongodb-org-shell mongocli \
  sqlcmd unixodbc-dev

# Download and install Oracle Instant Client and SQL*Plus
RUN URL_ORACLE_BASIC= && URL_ORACLE_SQLPLUS= && dpkgArch="$(dpkg --print-architecture)" \
  && case "${dpkgArch##*-}" in \
  amd64) URL_ORACLE_BASIC='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-basic-linux.x64-19.24.0.0.0dbru.zip';; \
  arm64) URL_ORACLE_BASIC='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-basic-linux.arm64-19.24.0.0.0dbru.zip';; \
  i386) URL_ORACLE_BASIC='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-basic-linux-19.24.0.0.0dbru.zip';; \
  *) echo "unsupported architecture"; exit 1 ;; \
  esac \
  && case "${dpkgArch##*-}" in \
  amd64) URL_ORACLE_SQLPLUS='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-sqlplus-linux.x64-19.24.0.0.0dbru.zip';; \
  arm64) URL_ORACLE_SQLPLUS='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-sqlplus-linux.arm64-19.24.0.0.0dbru.zip';; \
  i386) URL_ORACLE_SQLPLUS='https://download.oracle.com/otn_software/linux/instantclient/1924000/instantclient-sqlplus-linux-19.24.0.0.0dbru.zip';; \
  *) echo "unsupported architecture"; exit 1 ;; \
  esac \
  && mkdir -p /opt/oracle/instantclient_19_24 && \
  cd /opt/oracle && \
  wget -O instantclient-basic-linux.zip $URL_ORACLE_BASIC && \
  wget -O instantclient-sqlplus-linux.zip $URL_ORACLE_SQLPLUS && \
  unzip instantclient-basic-linux.zip && \
  rm -rf META-INF && \
  unzip instantclient-sqlplus-linux.zip && \
  rm instantclient-basic-linux.zip && \
  rm instantclient-sqlplus-linux.zip && \
  echo 'set markup csv on delimiter "\t" quote off\nset heading on echo off termout off\nset feedback off trimspool on' >> instantclient_19_24/glogin.sql && \
  cd /

# Configure environment variables
ENV PATH "/opt/oracle/instantclient_19_24:$PATH"
ENV LD_LIBRARY_PATH "/opt/oracle/instantclient_19_24:$LD_LIBRARY_PATH"

RUN curl -sL "https://dl.k8s.io/release/v1.22.1/bin/linux/amd64/kubectl" -o kubectl && \
  echo '78178a8337fc6c76780f60541fca7199f0f1a2e9c41806bded280a4a5ef665c9  kubectl' | sha256sum -c --ignore-missing --strict - && \
  chmod 755 kubectl && \
  mv kubectl /usr/local/bin/kubectl

RUN URL= && dpkgArch="$(dpkg --print-architecture)" \
  && case "${dpkgArch##*-}" in \
  amd64) URL="https://github.com/PostgREST/postgrest/releases/download/v$POSTGREST_VERSION/postgrest-v$POSTGREST_VERSION-linux-static-x64.tar.xz";; \
  arm64) URL="https://github.com/PostgREST/postgrest/releases/download/v$POSTGREST_VERSION/postgrest-v$POSTGREST_VERSION-ubuntu-aarch64.tar.xz";; \
  *) echo "unsupported architecture"; exit 1 ;; \
  esac \
  && curl -sL $URL -o postgrest.tar.xz && \
  tar -xf postgrest.tar.xz && rm -f postgrest.tar.xz && \
  mv postgrest /usr/local/bin/postgrest && \
  chmod 0755 /usr/local/bin/postgrest && \
  postgrest --version

COPY rootfs/usr/local/bin/mongosh /usr/local/bin/mongosh
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && \
  locale-gen
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

ENV PATH "/app:${PATH}"

ENTRYPOINT ["tini", "--"]
