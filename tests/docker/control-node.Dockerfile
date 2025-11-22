# Dockerfile for control node (runs Ansible and AnsiGo)
FROM ubuntu:24.04

# 安装必要的工具
RUN apt-get update && \
    apt-get install -y \
    python3 \
    python3-pip \
    python3-venv \
    openssh-client \
    curl \
    git \
    vim \
    && apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 安装 Ansible
RUN python3 -m pip install --break-system-packages ansible-core==2.16.0

# 安装 Go (用于构建 AnsiGo)
RUN curl -L https://go.dev/dl/go1.23.4.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/root/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# 创建工作目录
WORKDIR /workspace

# 设置 SSH 客户端配置
RUN mkdir -p /root/.ssh && \
    echo "Host *" >> /root/.ssh/config && \
    echo "    StrictHostKeyChecking no" >> /root/.ssh/config && \
    echo "    UserKnownHostsFile=/dev/null" >> /root/.ssh/config && \
    chmod 600 /root/.ssh/config

CMD ["/bin/bash"]
