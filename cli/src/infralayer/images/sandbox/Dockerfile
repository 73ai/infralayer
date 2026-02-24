FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

ENV TZ=UTC

# common utils
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    jq \
    openssh-client \
    make \
    unzip \
    gnupg \
    lsb-release \
    software-properties-common \
    ca-certificates \
    apt-transport-https \
    sudo \
    vim \
    less \
    tree \
    htop \
    net-tools \
    dnsutils \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*


# Install Google Cloud SDK with GKE auth plugin
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg \
    && apt-get update && apt-get install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl
ARG TARGETARCH
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl" \
    && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
    && rm kubectl

# Create workspace directory
WORKDIR /workspace

# Set default shell to bash
SHELL ["/bin/bash", "-c"]

# Default command - keep container alive
CMD ["tail", "-f", "/dev/null"]
